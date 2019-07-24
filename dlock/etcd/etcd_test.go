package etcd

import (
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"testing"
	"time"
)

func TestEtcdLock(t *testing.T) {
	var conf = clientv3.Config{
		Endpoints:   []string{"172.30.60.8:2379"},
		DialTimeout: 5 * time.Second,
	}
	l1 := &Dlock{
		Conf: conf,
		Ttl:  10,
		Key:  "lock",
	}

	l2 := &Dlock{
		Conf: conf,
		Ttl:  10,
		Key:  "lock",
	}

	go func() {
		err := l1.Acquire()
		if err != nil {
			fmt.Println("groutine1抢锁失败")
			fmt.Println(err)
			return
		}
		fmt.Println("groutine1抢锁成功")
		time.Sleep(10 * time.Second)
		defer l1.Release()
	}()

	go func() {
		err := l2.Acquire()
		if err != nil {
			fmt.Println("groutine2抢锁失败")
			fmt.Println(err)
			return
		}
		fmt.Println("groutine2抢锁成功")
		defer l2.Release()
	}()
	time.Sleep(30 * time.Second)
}
