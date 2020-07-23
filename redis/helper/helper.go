package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
)

// Job defines user updates
type Job struct {
	Name string      // name format: tag1=val1;tag2=val2 (i.e., user=test)
	Time int64       // time in seconds
	Desc interface{} // Used only for reporting
}

// Find a pending task to be processed.
// Jobs are locked before returning.
func Find(ctx context.Context, r *redis.Client, jobNS string, lockNS string, uuid string, timeout int) (jChan chan Job, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "FindJob")
	defer span.Finish()

	jChan = make(chan Job, 1)

	j, err := r.Eval(ctx, `
	redis.replicate_commands()
	local i = 0 
	local jobsNs = KEYS[1]
	local lockNs = ARGV[1]

	while 1 do
		local tab = redis.call( 'ZSCAN', jobsNs, i )
		i = tab[1]
		local set = tab[2]

		for j=1,#set,2 do			
			local ok = redis.call('SET', lockNs .. set[j], ARGV[2], 'NX', 'EX', ARGV[3])
			if ok then
				return {set[j], set[j+1]}
			end
		end
		
		if tonumber(i) == 0 then
			return 'none'
		end
	end

	return nil
	`, []string{jobNS}, lockNS, uuid, timeout).Result()

	// error in redis evaluation
	if err != nil {
		panic(fmt.Sprintf("cannot get job: '%v'", err))
	}

	// no job received
	if j == "none" {
		return
	}

	// extract job parameters
	jmap := j.([]interface{})
	n := jmap[0].(string)
	t := jmap[1].(string)
	tt, err := strconv.ParseInt(t, 0, 64)

	if err != nil {
		err = fmt.Errorf("cannot convert job time: '%v'", err)
		return
	}

	jChan <- Job{Name: n, Time: tt}
	return
}

// Unlock attempts to remove the lock on a key so long as the value matches.
// If the lock cannot be removed, either because the key has already expired or
// because the value was incorrect, an error will be returned.
func Unlock(ctx context.Context, r *redis.Client, lockNS string, key string, uuid string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UnlockJob")
	defer span.Finish()

	unlock, err := r.Eval(ctx, `
	redis.replicate_commands()
	local key = KEYS[1]
	if redis.call('GET', key) == ARGV[1] then
		return redis.call('DEL', key)
	else
		return 0
	end
	`, []string{lockNS + key}, uuid).Result()

	if err != nil {
		return false, err
	}
	return unlock.(int64) == 1, err
}

// Remove removes job from redis
func Remove(ctx context.Context, r *redis.Client, jobNS string, j Job) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RemoveJob")
	defer span.Finish()

	res, err := r.Eval(ctx, `
	redis.replicate_commands()
	local jobsNs = KEYS[1]
	local m = redis.call('ZSCAN', jobsNs, 0, 'MATCH', ARGV[1])
	local t = m[2][2]
	if t == ARGV[2] then
		return redis.call('ZREM', jobsNs, ARGV[1])
	else
		return 0
	end
	`, []string{jobNS}, j.Name, j.Time).Result()

	if err != nil {
		return false, err
	}

	return res.(int64) == 1, err
}

// ExLock sets the expiry of already owned lock
// TODO: test
func ExLock(ctx context.Context, r *redis.Client, key string, uuid string, timeout int) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ExLock")
	defer span.Finish()

	lock, err := r.Eval(ctx, `
	redis.replicate_commands()
	local ok = redis.call('SET', KEYS[1], ARGV[1], 'NX', 'EX', ARGV[2])
	if ok then
		return 1
	else 
		return 0
	end
	`, []string{key}, uuid, timeout).Result()

	if err != nil {
		return false, err
	}
	return lock.(int64) == 1, err
}

// Add adds the update to redis
func Add(ctx context.Context, r *redis.Client, jobNS string, j Job) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "AddJob")
	defer span.Finish()

	if j == (Job{}) {
		return fmt.Errorf("cannot add empty job")
	}
	z := redis.Z{Member: j.Name, Score: float64(j.Time)}
	_, err := r.ZAdd(ctx, jobNS, &z).Result()
	if err != nil {
		return err
	}
	return nil
}

// Set sets a state which expires. It uses Redis Set command.
func Set(ctx context.Context, r redis.Cmdable, ns string, name string, value interface{}, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Set")
	defer span.Finish()

	// persist fake state
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("could not marshal state: %v", err)
	}

	key := ns + ":" + name
	if err := r.Set(ctx, key, b, ttl).Err(); err != nil {
		return fmt.Errorf("could not set value: %v", err)
	}
	return nil
}

// Get gets user state from redis using the Get command.
func Get(ctx context.Context, r redis.Cmdable, ns string, name string, value interface{}) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Get")
	defer span.Finish()

	key := ns + ":" + name

	idx, err := r.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("cannot check state for '%v': %v", key, err)
	}
	if idx == 0 { // key does not exist
		return nil
	}

	str, err := r.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("cannot get value for '%v': %v", key, err)
	}

	err = json.Unmarshal([]byte(str), &value)
	if err != nil {
		return fmt.Errorf("cannot unmarshal value for '%v': %v", key, err)
	}
	return nil
}

// HSet sets a state using the HSet command.
func HSet(ctx context.Context, r redis.Cmdable, ns string, name string, value interface{}) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "HSet")
	defer span.Finish()

	// persist fake state
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("could not marshal state: %v", err)
	}

	_, err = r.HSet(ctx, ns, name, b).Result()
	if err != nil {
		return fmt.Errorf("could not save state: %v", err)
	}
	return nil
}

// HGet gets user state from redis using the HGet command.
func HGet(ctx context.Context, r redis.Cmdable, ns string, name string, value interface{}) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "HGet")
	defer span.Finish()

	ok, err := r.HExists(ctx, ns, name).Result()
	if err != nil {
		err = fmt.Errorf("cannot check state for '%v': %v", name, err)
		return
	} else if !ok {
		return
	}

	str, err := r.HGet(ctx, ns, name).Result()
	if err != nil {
		err = fmt.Errorf("cannot get state for '%v': %v", name, err)
		return
	}

	err = json.Unmarshal([]byte(str), &value)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal state for '%v': %v", name, err)
		return
	}
	return
}
