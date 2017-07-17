package kvdb_redis

import (
	"github.com/garyburd/redigo/redis"
	"github.com/google/btree"
	"github.com/pkg/errors"
	"github.com/xiaonanln/goworld/gwlog"
	. "github.com/xiaonanln/goworld/kvdb/types"
)

const (
	keyPrefix = "_KV_"
)

type redisKVDB struct {
	c       redis.Conn
	keyTree *btree.BTree
}

type keyTreeItem struct {
	key string
}

func (ki keyTreeItem) Less(_other btree.Item) bool {
	return ki.key < _other.(keyTreeItem).key
}

func OpenRedisKVDB(host string) (*redisKVDB, error) {
	c, err := redis.Dial("tcp", host)
	if err != nil {
		return nil, errors.Wrap(err, "redis dail failed")
	}

	db := &redisKVDB{
		c:       c,
		keyTree: btree.New(2),
	}
	if err := db.initialize(); err != nil {
		panic(errors.Wrap(err, "redis kvdb initialize failed"))
	}

	return db, nil
}

func (db *redisKVDB) initialize() error {
	r, err := redis.Values(db.c.Do("SCAN", "0", "MATCH", keyPrefix+"*", "COUNT", 10000))
	if err != nil {
		return err
	}
	for {
		nextCursor := r[0]
		keys, err := redis.Strings(r[1], nil)
		if err != nil {
			return err
		}
		gwlog.Info("SCAN: %v, nextcursor=%s", keys, string(nextCursor.([]byte)))
		for _, key := range keys {
			key := key[len(keyPrefix):]
			db.keyTree.ReplaceOrInsert(keyTreeItem{key})
		}

		if db.isZeroCursor(nextCursor) {
			break
		}
		r, err = redis.Values(db.c.Do("SCAN", nextCursor))
	}
	return nil
}

func (db *redisKVDB) isZeroCursor(c interface{}) bool {
	return string(c.([]byte)) == "0"
}

func (db *redisKVDB) Get(key string) (val string, err error) {
	r, err := db.c.Do("GET", keyPrefix+key)
	if err != nil {
		return "", err
	}
	if r == nil {
		return "", nil
	} else {
		return string(r.([]byte)), err
	}
}

func (db *redisKVDB) Put(key string, val string) error {
	_, err := db.c.Do("SET", keyPrefix+key, val)
	return err
}

type RedisKVIterator struct {
	db       *redisKVDB
	beginKey string
}

func (db *redisKVDB) Find(key string) Iterator {
	db.keyTree.AscendGreaterOrEqual(keyTreeItem{key}, func(ki btree.Item) bool {
	})
	return nil
}
