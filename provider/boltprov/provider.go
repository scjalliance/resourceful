package boltprov

import (
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/scjalliance/resourceful/lease"
)

const (
	// ResourcefulBucket is the default name of the resourceful boltdb bucket in
	// which the provider stores data.
	ResourcefulBucket = "resourceful"
	// LeaseBucket is the name of the resourceful lease bucket.
	LeaseBucket = "lease"
)

// Provider provides boltdb-backed lease management.
type Provider struct {
	db   *bolt.DB
	root []byte
}

// New returns a new memory provider.
func New(db *bolt.DB) *Provider {
	return &Provider{
		db:   db,
		root: []byte(ResourcefulBucket),
	}
}

// Close releases any resources consumed by the provider.
func (p *Provider) Close() error {
	return p.db.Close()
}

// ProviderName returns the name of the provider.
func (p *Provider) ProviderName() string {
	return "bolt db"
}

// LeaseResources returns all of the resources with lease data.
func (p *Provider) LeaseResources() (resources []string, err error) {
	err = p.db.View(func(btx *bolt.Tx) error {
		root := btx.Bucket(p.root)
		if root == nil {
			return nil
		}

		container := root.Bucket([]byte(LeaseBucket))
		if container == nil {
			return nil
		}

		c := container.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if v != nil {
				resources = append(resources, string(k))
			}
		}

		return nil
	})
	return
}

// LeaseView returns the current revision and lease set for the resource.
func (p *Provider) LeaseView(resource string) (revision uint64, leases lease.Set, err error) {
	err = p.db.View(func(btx *bolt.Tx) error {
		root := btx.Bucket(p.root)
		if root == nil {
			return nil
		}

		container := root.Bucket([]byte(LeaseBucket))
		if container == nil {
			return nil
		}

		revision = container.Sequence()

		data := container.Get([]byte(resource))
		if data == nil {
			return nil
		}

		return json.Unmarshal(data, &leases)
	})
	return
}

// LeaseCommit will attempt to apply the operations described in the lease
// transaction.
func (p *Provider) LeaseCommit(tx *lease.Tx) error {
	ops := tx.Ops()
	if len(ops) == 0 {
		// Nothing to commit
		return nil
	}

	leases := tx.Leases()

	return p.db.Update(func(btx *bolt.Tx) error {
		root, err := btx.CreateBucketIfNotExists(p.root)
		if err != nil {
			return err
		}

		container, err := root.CreateBucketIfNotExists([]byte(LeaseBucket))
		if err != nil {
			return err
		}

		if container.Sequence() != tx.Revision() {
			return errors.New("Unable to commit lease transaction due to opportunistic lock conflict")
		}

		_, err = container.NextSequence()
		if err != nil {
			return err
		}

		key := []byte(tx.Resource())
		if len(leases) == 0 {
			return container.Delete(key)
		}

		value, err := json.Marshal(leases)
		if err != nil {
			return err
		}
		return container.Put(key, value)
	})
}
