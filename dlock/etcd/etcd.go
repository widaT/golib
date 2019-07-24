package etcd

import (
	"context"
	"errors"
	"go.etcd.io/etcd/clientv3"
)

var GetLock_Error = errors.New("get lock failed")

type Dlock struct {
	Ttl     int64              //租约时间
	Conf    clientv3.Config    //etcd集群配置
	Key     string             //etcd的key
	cancel  context.CancelFunc //关闭续租的func
	lease   clientv3.Lease
	leaseID clientv3.LeaseID
	txn     clientv3.Txn
}

func (l *Dlock) init() error {
	var err error
	var ctx context.Context
	client, err := clientv3.New(l.Conf)
	if err != nil {
		return err
	}
	l.txn = clientv3.NewKV(client).Txn(context.Background())
	l.lease = clientv3.NewLease(client)
	leaseResp, err := l.lease.Grant(context.Background(), l.Ttl)
	if err != nil {
		return err
	}
	ctx, l.cancel = context.WithCancel(context.Background())
	l.leaseID = leaseResp.ID
	_, err = l.lease.KeepAlive(ctx, l.leaseID)
	return err
}

func (l *Dlock) Acquire() error {
	err := l.init()
	if err != nil {
		return err
	}
	l.txn.If(clientv3.Compare(clientv3.CreateRevision(l.Key), "=", 0)).
		Then(clientv3.OpPut(l.Key, "", clientv3.WithLease(l.leaseID))).
		Else()
	txnResp, err := l.txn.Commit()
	if err != nil {
		return err
	}
	if !txnResp.Succeeded {
		return GetLock_Error
	}
	return nil
}
func (l *Dlock) Release() {
	l.cancel()
	l.lease.Revoke(context.Background(), l.leaseID)
}
