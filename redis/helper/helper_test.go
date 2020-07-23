package helper

import (
	"context"
	"flag"
	"os"
	"reflect"
	"testing"

	minisentinel "github.com/Bose/minisentinel"
	miniredis "github.com/alicebob/miniredis/v2"
	redis "github.com/go-redis/redis"
)

var red *redis.Client
var jobNS = "test:jobs"
var lockNS = "test:lock:"

func TestFind(t *testing.T) {
	ctx := context.Background()

	type args struct {
		r       *redis.Client
		jobNS   string
		lockNS  string
		uuid    string
		timeout int
	}
	tests := []struct {
		name      string
		args      args
		wantJChan chan Job
		wantErr   bool
	}{
		{
			name: "TestFind1",
			args: args{
				r:       red,
				jobNS:   jobNS,
				lockNS:  lockNS,
				uuid:    "test",
				timeout: 5,
			},
			wantJChan: make(chan Job, 1),
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotJChan, err := Find(ctx, tt.args.r, tt.args.jobNS, tt.args.lockNS, tt.args.uuid, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !jobChanEquals(gotJChan, tt.wantJChan) {
				t.Errorf("Find() = %v, want %v", gotJChan, tt.wantJChan)
			}
		})
	}
}

func TestUnlock(t *testing.T) {
	ctx := context.Background()

	type args struct {
		r      *redis.Client
		lockNS string
		key    string
		uuid   string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Unlock(ctx, tt.args.r, tt.args.lockNS, tt.args.key, tt.args.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Unlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	ctx := context.Background()

	type args struct {
		r     *redis.Client
		jobNS string
		j     Job
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Remove(ctx, tt.args.r, tt.args.jobNS, tt.args.j)
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Remove() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExLock(t *testing.T) {
	type args struct {
		r       *redis.Client
		key     string
		uuid    string
		timeout int
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExLock(tt.args.r, tt.args.key, tt.args.uuid, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExLock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	ctx := context.Background()

	type args struct {
		r     *redis.Client
		jobNS string
		j     Job
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Add(ctx, tt.args.r, tt.args.jobNS, tt.args.j); (err != nil) != tt.wantErr {
				t.Errorf("AddJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcess(t *testing.T) {
	ctx := context.Background()

	t.Run("AddJob", func(t *testing.T) {
		type args struct {
			r     *redis.Client
			jobNS string
			j     Job
		}
		tests := []struct {
			name    string
			args    args
			wantErr bool
		}{
			{ // Add test cases.
				name: "Empty job ",
				args: args{
					r:     red,
					jobNS: jobNS,
					j:     Job{},
				},
				wantErr: true,
			},
			{ // Add test cases.
				name: "Add user=test",
				args: args{
					r:     red,
					jobNS: jobNS,
					j:     Job{Name: "user=test", Time: 0},
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := Add(ctx, tt.args.r, tt.args.jobNS, tt.args.j); (err != nil) != tt.wantErr {
					t.Errorf("AddJob() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	// -----------------------------------------

	t.Run("Find", func(t *testing.T) {
		type args struct {
			r       *redis.Client
			jobNS   string
			lockNS  string
			uuid    string
			timeout int
		}
		tests := []struct {
			name      string
			args      args
			wantJChan chan Job
			wantErr   bool
		}{
			{ // Add test cases.
				name: "Find user=test",
				args: args{
					r:       red,
					jobNS:   jobNS,
					lockNS:  lockNS,
					uuid:    "test1",
					timeout: 5,
				},
				wantJChan: func() (j chan Job) {
					j = make(chan Job, 1)
					j <- Job{Name: "user=test", Time: 0}
					return
				}(),
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gotJChan, err := Find(ctx, tt.args.r, tt.args.jobNS, tt.args.lockNS, tt.args.uuid, tt.args.timeout)
				if (err != nil) != tt.wantErr {
					t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !jobChanEquals(gotJChan, tt.wantJChan) {
					t.Errorf("Find() = %v, want %v", gotJChan, tt.wantJChan)
				}
			})
		}
	})

	// -----------------------------------------

	t.Run("Unlock", func(t *testing.T) {
		type args struct {
			r      *redis.Client
			lockNS string
			key    string
			uuid   string
		}
		tests := []struct {
			name    string
			args    args
			want    bool
			wantErr bool
		}{
			// Add test cases.
			{
				name: "Unlock with a wrong uuid",
				args: args{
					r:      red,
					lockNS: lockNS,
					key:    "user=test",
					uuid:   "test2",
				},
				want:    false,
				wantErr: false,
			},
			{
				name: "Unlock with a right uuid",
				args: args{
					r:      red,
					lockNS: lockNS,
					key:    "user=test",
					uuid:   "test1",
				},
				want:    true,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := Unlock(ctx, tt.args.r, tt.args.lockNS, tt.args.key, tt.args.uuid)
				if (err != nil) != tt.wantErr {
					t.Errorf("Unlock() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("Unlock() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	// -----------------------------------------

	t.Run("Remove", func(t *testing.T) {
		type args struct {
			r     *redis.Client
			jobNS string
			j     Job
		}
		tests := []struct {
			name    string
			args    args
			want    bool
			wantErr bool
		}{
			// Add test cases.
			{
				name: "Remove with non-existent user",
				args: args{
					r:     red,
					jobNS: jobNS,
					j:     Job{Name: "user=testxxx", Time: -1},
				},
				want:    false,
				wantErr: false,
			},
			{
				name: "Remove with outdated score",
				args: args{
					r:     red,
					jobNS: jobNS,
					j:     Job{Name: "user=test", Time: -1},
				},
				want:    false,
				wantErr: false,
			},
			{
				name: "Remove with outdated score",
				args: args{
					r:     red,
					jobNS: jobNS,
					j:     Job{Name: "user=test", Time: 0},
				},
				want:    true,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := Remove(ctx, tt.args.r, tt.args.jobNS, tt.args.j)
				if (err != nil) != tt.wantErr {
					t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("Remove() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func jobChanEquals(c1 chan Job, c2 chan Job) bool {

	if len(c1) == 0 && len(c2) == 0 {
		return true
	}
	for len(c1) > 0 {
		v1 := <-c1
		v2 := <-c2
		ok := reflect.DeepEqual(v1, v2)
		if !ok {
			return false
		}
	}
	return true
}

func TestMain(m *testing.M) {
	// var err error
	flag.Parse()

	//* init redis
	// create standalone redis
	minired, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer minired.Close()

	//* run with standalone redis
	red = redis.NewClient(&redis.Options{
		Addr: minired.Addr(),
	})
	defer red.Close()

	res := m.Run()
	if res != 0 {
		os.Exit(res)
	}

	//* run with redis sentinel
	// create redis sentinel
	s := minisentinel.NewSentinel(minired, minisentinel.WithReplica(minired))
	s.Start()
	defer s.Close()

	red = redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    s.MasterInfo().Name,
		SentinelAddrs: []string{s.Addr()},
		MaxRetries:    5,
	})
	defer red.Close()

	os.Exit(m.Run())
}
