package siapubaccess

import (
	"io"
	"sync"

	"github.com/EvilRedHorse/go-pubaccess"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/objfile"
	"github.com/quorumcontrol/dgit/storage"
	"go.uber.org/zap"
)

type uploadJob struct {
	o      plumbing.EncodedObject
	result chan string
	err    chan error
}

type downloadJob struct {
	link   string
	result chan plumbing.EncodedObject
	err    chan error
}

type Pubaccess struct {
	sync.RWMutex

	uploaderCount      int
	downloaderCount    int
	uploadersStarted   bool
	downloadersStarted bool
	uploadJobs         chan *uploadJob
	downloadJobs       chan *downloadJob

	log *zap.SugaredLogger
}

func InitPubaccess(uploaderCount, downloaderCount int) *Pubaccess {
	return &Pubaccess{
		uploaderCount:   uploaderCount,
		downloaderCount: downloaderCount,
		uploadJobs:      make(chan *uploadJob),
		downloadJobs:    make(chan *downloadJob),
		log:             log.Named("net"),
	}
}

func (s *Pubaccess) uploadObject(o plumbing.EncodedObject) (string, error) {
	buf, err := storage.ZlibBufferForObject(o)
	if err != nil {
		return "", err
	}

	uploadData := make(pubaccess.UploadData)
	uploadData[o.Hash().String()] = buf

	link, err := pubaccess.Upload(uploadData, pubaccess.DefaultUploadOptions)

	return link, nil
}

func (s *Pubaccess) startUploader() {
	for j := range s.uploadJobs {
		s.log.Debugf("uploading %s to Public Portals", j.o.Hash())
		link, err := s.uploadObject(j.o)
		if err != nil {
			j.err <- err
			continue
		}
		j.result <- link
	}
}

func (s *Pubaccess) startUploaders() {
	s.log.Debugf("starting %d uploader(s)", s.uploaderCount)

	for i := 0; i < s.uploaderCount; i++ {
		go s.startUploader()
	}
}

func (s *Pubaccess) UploadObject(o plumbing.EncodedObject) (chan string, chan error) {
	s.Lock()
	if !s.uploadersStarted {
		s.startUploaders()
		s.uploadersStarted = true
	}
	s.Unlock()

	result := make(chan string)
	err := make(chan error)

	s.uploadJobs <- &uploadJob{
		o:      o,
		result: result,
		err:    err,
	}

	return result, err
}

func (s *Pubaccess) downloadObject(link string) (plumbing.EncodedObject, error) {
	objData, err := pubaccess.Download(link, pubaccess.DefaultDownloadOptions)
	if err != nil {
		return nil, err
	}

	o := &plumbing.MemoryObject{}

	reader, err := objfile.NewReader(objData)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	objType, size, err := reader.Header()
	if err != nil {
		return nil, err
	}

	o.SetType(objType)
	o.SetSize(size)

	if _, err = io.Copy(o, reader); err != nil {
		return nil, err
	}

	return o, nil
}

func (s *Pubaccess) startDownloader() {
	for j := range s.downloadJobs {
		s.log.Debugf("downloading %s from Public Portals", j.link)
		o, err := s.downloadObject(j.link)
		if err != nil {
			j.err <- err
			continue
		}
		j.result <- o
	}
}

func (s *Pubaccess) startDownloaders() {
	s.log.Debugf("starting %d downloader(s)", s.downloaderCount)

	for i := 0; i < s.downloaderCount; i++ {
		go s.startDownloader()
	}
}

func (s *Pubaccess) DownloadObject(link string) (chan plumbing.EncodedObject, chan error) {
	s.Lock()
	if !s.downloadersStarted {
		s.startDownloaders()
		s.downloadersStarted = true
	}
	s.Unlock()

	result := make(chan plumbing.EncodedObject)
	err := make(chan error)

	s.downloadJobs <- &downloadJob{
		link:   link,
		result: result,
		err:    err,
	}

	return result, err
}
