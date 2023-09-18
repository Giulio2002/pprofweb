package pprofweb

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"gfx.cafe/util/go/generic"
	"tuxpa.in/a/pprofweb/flagset"
	"github.com/go-chi/chi/v5"
	"github.com/google/pprof/driver"
	"github.com/rs/xid"
	"github.com/spf13/afero"
)

type Config struct {
	Expire        time.Duration
	MaxUploadSize int
}

type Server struct {
	fs afero.Fs

	instances generic.Map[xid.ID, *Instance]

	Config
}

func NewServer(fs afero.Fs, c Config) *Server {
	srv := afero.NewCacheOnReadFs(fs, afero.NewMemMapFs(), 24*time.Hour)
	s := &Server{
		fs:     srv,
		Config: c,
	}

	if s.MaxUploadSize == 0 {
		s.MaxUploadSize = 32 << 20
	}

	return s
}

// handler returns a handler that servers the pprof web UI.
func (s *Server) HandleHTTP() func(chi.Router) {
	return func(r chi.Router) {
		r.HandleFunc("/", rootHandler)
		r.HandleFunc("/upload", s.HandleUpload)
		r.HandleFunc("/{xid}", s.ServeInstance)
		r.HandleFunc("/{xid}/*", s.ServeInstance)

	}
}

func (s *Server) ServeInstance(w http.ResponseWriter, r *http.Request) {
	err := s.serveInstance(w, r)
	if err != nil {
		http.Error(w, "instance not found: "+err.Error(), http.StatusNotFound)
		return
	}
}

func (s *Server) serveInstance(w http.ResponseWriter, r *http.Request) error {
	log.Printf("serveInstance %s %s", r.Method, r.URL.String())
	sid := chi.URLParam(r, "xid")
	if sid == "" {
		return errors.New("no id sent")
	}
	id, err := xid.FromString(sid)
	if err != nil {
		return errors.New("invalid id")
	}
	inst, err := s.GetInstance(id)
	if err != nil {
		return errors.New("instance not found: " + err.Error())
	}
	inst.ServeHTTP(w, r)
	return nil
}

func (s *Server) NewInstance(xs []byte) (*Instance, error) {
	// save instance
	id := xid.New()
	i := &Instance{
		id:  id,
		dat: xs,
	}

	file, err := s.fs.Create(id.String())
	if err != nil {
		return nil, err
	}
	_, err = file.Write(xs)
	if err != nil {
		file.Close()
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	err = i.startServer()
	if err != nil {
		return nil, err
	}
	s.instances.Store(id, i)
	return i, nil
}

func (s *Server) GetInstance(id xid.ID) (*Instance, error) {
	if i, ok := s.instances.Load(id); ok {
		return i, nil
	}
	if s.Expire > 0 {
		if id.Time().After(time.Now().Add(s.Expire)) {
			s.fs.Remove(id.String())
			return nil, fmt.Errorf("profile expired")
		}
	}
	bts, err := afero.ReadFile(s.fs, id.String())
	if err != nil {
		return nil, err
	}
	i := &Instance{id: id, dat: bts}
	err = i.startServer()
	if err != nil {
		return nil, err
	}
	s.instances.Store(id, i)
	return i, nil
}

func (s *Server) HandleUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("uploadHandler %s %s", r.Method, r.URL.String())
	if r.Method != http.MethodPost {
		http.Error(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}
	err := s.handleUpload(w, r)
	if err != nil {
		log.Printf("upload error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseMultipartForm(int64(s.MaxUploadSize)); err != nil {
		return err
	}
	uploadedFile, _, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer uploadedFile.Close()
	bts, err := io.ReadAll(uploadedFile)
	if err != nil {
		return err
	}

	instance, err := s.NewInstance(bts)
	if err != nil {
		return err
	}
	for _, h := range r.Header.Values("Accept") {
		if strings.Contains(h, "text/html") {
			http.Redirect(w, r, instance.id.String()+"/", http.StatusSeeOther)
			return nil
		}
	}
	w.Write([]byte(instance.id.String()))
	return nil
}

type Instance struct {
	id  xid.ID
	dat []byte

	handler http.Handler

	mu sync.RWMutex
}

func (i *Instance) startServer() error {
	file, err := os.CreateTemp("", i.id.String())
	if err != nil {
		return err
	}
	file.Write(i.dat)
	flags := &flagset.Set{
		Argz: []string{"-http=localhost:0", "-no_browser", file.Name()},
	}
	options := &driver.Options{
		Flagset:    flags,
		HTTPServer: i.startHTTP,
	}
	if err := driver.PProf(options); err != nil {
		return err
	}
	return nil
}
func (i *Instance) startHTTP(args *driver.HTTPServerArgs) error {
	r := chi.NewRouter()
	for pattern, handler := range args.Handlers {
		jpat := "/" + path.Join(i.id.String(), pattern)
		r.Handle(jpat, handler)
		r.Handle(jpat+"/", handler)
	}
	i.handler = r
	return nil
}

func (i *Instance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("serveInstanceHTTP %s %s", i.id, r.URL.String())
	i.handler.ServeHTTP(w, r)
}

// Mostly copied from https://github.com/google/pprof/blob/master/internal/driver/flags.go
type pprofFlags struct {
	args  []string
	s     flag.FlagSet
	usage []string
}
