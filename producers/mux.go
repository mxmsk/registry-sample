package producers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

// Producer defines a plugin interface for ServeMux.
// Taking a named source Producer provides output in a concrete format.
type Producer interface {
	// HTML generates output to display data as a web page.
	HTML(w io.Writer, name string) error
}

// ServeMux maps producers to HTTP requests by implementing http.Handler.
// Producer is matched by the first segment of URL following the baseURL.
type ServeMux struct {
	baseURL   string
	producers map[string]Producer
	mu        sync.Mutex
}

// NewServeMux creates and initializes a new instance of ServeMux.
func NewServeMux(baseURL string) *ServeMux {
	return &ServeMux{
		baseURL:   baseURL,
		producers: make(map[string]Producer),
	}
}

// AddProducer adds the specified Producer and maps it to the specified
// key. Notice that key must be unique and can't be empty.
func (mux *ServeMux) AddProducer(key string, p Producer) error {
	if p == nil {
		return fmt.Errorf("Trying to add nil Producer with key %s", key)
	}
	if len(strings.TrimSpace(key)) == 0 {
		// Maybe in a future we could support default Producer
		// by allowing to register it with an empty string.
		return errors.New("Producer's key cannot be blank")
	}

	mux.mu.Lock()
	defer mux.mu.Unlock()

	if _, exists := mux.producers[key]; exists {
		return fmt.Errorf("Another Producer with key %s is already registered", key)
	}
	mux.producers[key] = p
	return nil
}

// ServeHTTP handles HTTP requests by transferring them to registered Producers.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			http.Error(w, "Unexpected error occured", http.StatusInternalServerError)
			mux.log("panic", r)
		}
	}()

	rel := r.URL.Path[len(mux.baseURL):]
	segs := strings.Split(rel, "/")
	if len(segs) != 2 {
		http.NotFound(w, r)
		return
	}

	pk := segs[0]
	name := segs[1]

	mux.mu.Lock()
	p, ok := mux.producers[pk]
	mux.mu.Unlock()

	if !ok {
		http.Error(w, fmt.Sprintf("%s is not supported", pk), http.StatusNotImplemented)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf("%s is not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}

	if err := p.HTML(w, name); err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Can't produce output", http.StatusInternalServerError)
		mux.log("error", err)
	}
}

func (mux *ServeMux) log(prefix string, v ...interface{}) {
	prefix = fmt.Sprintf("[%s]", strings.ToUpper(prefix))
	v = append([]interface{}{prefix}, v...)
	log.Println(v...)
}
