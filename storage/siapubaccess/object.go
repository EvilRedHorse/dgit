package siapubaccess

import (
	"fmt"
	"io"
	"strings"
	"sync"

	format "github.com/ipfs/go-ipld-format"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"

	"github.com/quorumcontrol/dgit/storage"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/chaintree/chaintree"
	"go.uber.org/zap"
)

const TupeloTxnBatchSize = 75

var log = logging.Logger("decentragit.storage.siapubaccess")

type ObjectStorage struct {
	*storage.ChaintreeObjectStorage
	log    *zap.SugaredLogger
	pubaccess *Pubaccess
}

var _ storer.EncodedObjectStorer = (*ObjectStorage)(nil)
var _ storer.PackfileWriter = (*ObjectStorage)(nil)
var _ storer.Transactioner = (*ObjectStorage)(nil)

func NewObjectStorage(config *storage.Config) storer.EncodedObjectStorer {
	did := config.ChainTree.MustId()
	return &ObjectStorage{
		&storage.ChaintreeObjectStorage{config},
		log.Named(did[len(did)-6:]),
		InitPubaccess(4, 1),
	}
}

type PublinkStore map[plumbing.Hash]string

type TemporalStorage struct {
	sync.RWMutex
	uploadWaitGroup sync.WaitGroup

	log      *zap.SugaredLogger
	publinks PublinkStore
	publink  *Pubaccess
}

type ChaintreeLinkStorage struct {
	log *zap.SugaredLogger
	*storage.Config
}

func NewTemporalStorage() *TemporalStorage {
	return &TemporalStorage{
		log:      log.Named("pubaccess-temporal"),
		publinks: make(PublinkStore),
		publink:   InitPubaccess(4, 1),
	}
}

func (ts *TemporalStorage) SetPublink(h plumbing.Hash, link string) {
	ts.Lock()
	defer ts.Unlock()

	ts.publinks[h] = link
}

func (ts *TemporalStorage) Publinks() PublinkStore {
	sls := make(PublinkStore)

	ts.RLock()
	defer ts.RUnlock()

	for h, l := range ts.publinks {
		sls[h] = l
	}

	return sls
}

func NewChaintreeLinkStorage(config *storage.Config) *ChaintreeLinkStorage {
	did := config.ChainTree.MustId()
	return &ChaintreeLinkStorage{
		log.Named(did[len(did)-6:]),
		config,
	}
}

func uploadObjectToPubaccess(s *Pubaccess, o plumbing.EncodedObject) (string, error) {
	resultC, errC := s.UploadObject(o)

	select {
	case err := <-errC:
		return "", err
	case link := <-resultC:
		return link, nil
	}
}

func (ts *TemporalStorage) SetEncodedObject(o plumbing.EncodedObject) (plumbing.Hash, error) {
	ts.log.Debugf("uploading %s to Public Portals", o.Hash())

	objHash := o.Hash()

	ts.uploadWaitGroup.Add(1)
	go func() {
		link, err := uploadObjectToPubaccess(ts.pubaccess, o)
		if err != nil {
			ts.log.Errorf("object %s upload failed: %w", objHash, err)
			return
		}

		ts.SetPublink(objHash, link)

		ts.uploadWaitGroup.Done()
	}()

	return objHash, nil
}

func downloadObjectFromPubaccess(s *Pubaccess, link string) (plumbing.EncodedObject, error) {
	resultC, errC := s.DownloadObject(link)

	select {
	case err := <-errC:
		return nil, err
	case o := <-resultC:
		return o, nil
	}
}

type ObjectTransaction struct {
	temporal *TemporalStorage
	storage  *ChaintreeLinkStorage
	log      *zap.SugaredLogger
}

var _ storer.Transaction = (*ObjectTransaction)(nil)

func (s *ObjectStorage) Begin() storer.Transaction {
	ts := NewTemporalStorage()
	ls := NewChaintreeLinkStorage(s.Config)
	return &ObjectTransaction{
		// NB: Currently TemporalStorage uploads objects to
		// public portals as they are added to the txn. This makes sense while it's
		// free, but perhaps less so once it isn't. It still might make sense
		// perf-wise, but you'd want to clean up on Rollback / error to stop
		// paying for those uploads.
		temporal: ts,
		storage:  ls,
		log:      s.log.Named("object-transaction"),
	}
}

func (ot *ObjectTransaction) SetEncodedObject(o plumbing.EncodedObject) (plumbing.Hash, error) {
	ot.log.Debugf("added object %s to transaction", o.Hash())
	return ot.temporal.SetEncodedObject(o)
}

func (ot *ObjectTransaction) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	ot.log.Errorf("ObjectTransaction.EncodedObject is a stub to satisfy the interface; don't call it")
	return &plumbing.MemoryObject{}, nil
}

func (ot *ObjectTransaction) Commit() error {
	ot.log.Debugf("committing transaction")

	var tupeloTxns []*transactions.Transaction

	// make sure all pending uploads have completed and set their publinks
	ot.log.Debugf("waiting for all Public Portal uploads to complete")
	ot.temporal.uploadWaitGroup.Wait()
	ot.log.Debugf("Public Portal uploads complete")

	publinks := ot.temporal.Publinks()

	for h, link := range publinks {
		txn, err := setLinkTxn(h, strings.Replace(link, "scp://", "did:scp:", 1))
		if err != nil {
			return err
		}

		tupeloTxns = append(tupeloTxns, txn)
	}

	if len(publinks) > 0 {
		txnBatch := make([]*transactions.Transaction, 0)
		lastIdx := len(tupeloTxns) - 1
		for i, t := range tupeloTxns {
			batchIdx := (i + 1) % TupeloTxnBatchSize
			txnBatch = append(txnBatch, t)
			if batchIdx == 0 || i == lastIdx {
				ot.log.Debugf("saving %d Publinks in transaction to repo chaintree", len(txnBatch))
				_, err := ot.storage.Tupelo.PlayTransactions(ot.storage.Ctx, ot.storage.ChainTree, ot.storage.PrivateKey, txnBatch)
				if err != nil {
					return err
				}
				txnBatch = make([]*transactions.Transaction, 0)
			}
		}
	}

	return nil
}

func setLinkTxn(h plumbing.Hash, link string) (*transactions.Transaction, error) {
	writePath := storage.ObjectWritePath(h)

	txn, err := chaintree.NewSetDataTransaction(writePath, link)
	if err != nil {
		return nil, err
	}

	return txn, nil
}

func (ot *ObjectTransaction) Rollback() error {
	ot.log.Debugf("rolling back transaction")
	ot.temporal = nil
	return nil
}

func (s *ObjectStorage) PackfileWriter() (io.WriteCloser, error) {
	s.log.Debug("packfile writer requested")
	return storage.NewPackWriter(s), nil
}

func (s *ObjectStorage) SetEncodedObject(o plumbing.EncodedObject) (plumbing.Hash, error) {
	s.log.Debugf("saving %s with type %s", o.Hash().String(), o.Type().String())

	if o.Type() == plumbing.OFSDeltaObject || o.Type() == plumbing.REFDeltaObject {
		return plumbing.ZeroHash, plumbing.ErrInvalidType
	}

	s.log.Debugf("uploading %s to Public Portals", o.Hash().String())
	link, err := uploadObjectToPubaccess(s.pubaccess, o)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	publink := strings.TrimPrefix(link, "scp://")
	objDid := strings.Join([]string{"did", "scp", publink}, ":")

	tx, err := setLinkTxn(o.Hash(), objDid)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	_, err = s.Tupelo.PlayTransactions(s.Ctx, s.ChainTree, s.PrivateKey, []*transactions.Transaction{tx})
	if err != nil {
		return plumbing.ZeroHash, err
	}

	return o.Hash(), nil
}

func (s *ObjectStorage) HasEncodedObject(h plumbing.Hash) (err error) {
	if _, err := s.EncodedObject(plumbing.AnyObject, h); err != nil {
		return err
	}
	return nil
}

func (s *ObjectStorage) EncodedObjectSize(h plumbing.Hash) (size int64, err error) {
	o, err := s.EncodedObject(plumbing.AnyObject, h)
	if err != nil {
		return 0, err
	}
	return o.Size(), nil
}

func (s *ObjectStorage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	s.log.Debugf("fetching %s with type %s", h.String(), t.String())

	path := storage.ObjectReadPath(h)
	valUncast, _, err := s.ChainTree.ChainTree.Dag.Resolve(s.Ctx, path)
	if err == format.ErrNotFound {
		s.log.Debugf("%s not found in chaintree at path %s", h, path)
		return nil, plumbing.ErrObjectNotFound
	}
	if err != nil {
		s.log.Errorf("chaintree resolve error for %s: %w", h, err)
		return nil, err
	}
	if valUncast == nil {
		s.log.Debugf("%s was nil in chaintree at path %s", h, path)
		return nil, plumbing.ErrObjectNotFound
	}

	// TODO: Read these in higher-level code and delegate decoding to whichever
	//  object storage system is specified in the did:storer: prefix
	objDid, ok := valUncast.(string)
	if !ok {
		s.log.Errorf("object DID should be a string; was a %T instead", valUncast)
		return nil, plumbing.ErrObjectNotFound
	}
	if !strings.HasPrefix(objDid, "did:scp:") {
		s.log.Errorf("object DID %s should start with did:scp:", objDid)
		return nil, plumbing.ErrObjectNotFound
	}

	link := strings.Replace(objDid, "did:scp:", "scp://", 1)

	s.log.Debugf("downloading %s from Public Portals at %s", h, link)
	o, err := downloadObjectFromPubaccess(s.pubaccess, link)
	if err != nil {
		err = fmt.Errorf("could not download object %s from Public Portals at %s: %w", h.String(), link, err)
		s.log.Errorf(err.Error())
		return nil, err
	}

	if plumbing.AnyObject != t && o.Type() != t {
		s.log.Debugf("%s not found, mismatched types, expected %s, got %s", h, t, o.Type())
		return nil, plumbing.ErrObjectNotFound
	}

	return o, nil
}

func (s *ObjectStorage) IterEncodedObjects(t plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	return storage.NewEncodedObjectIter(s, t), nil
}
