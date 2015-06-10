package leak

import(
"runtime"
"sync"
"sort"
"time"
"fmt"
"net/http"

"github.com/golang/glog"
)

type trackedObject struct {
	name string
	label string
	timestamp time.Time
	tracked bool
}

var objects map[string]*trackedObject
var objectsLock sync.Mutex

func init() {
	objects = make(map[string]*trackedObject, 1024 * 1024)
}

func Track(name,label string, obj interface{}) {
	objectsLock.Lock()
	ptr := fmt.Sprintf("%p", obj)
	_, ok := objects[ptr]
	if ok {
		panic(fmt.Sprintf("Already tracking object %q: %+v", name, obj))
	}
	objects[ptr] = &trackedObject{
		name: name,
		label: label,
		timestamp: time.Now(),
		tracked: true,
	}
	runtime.SetFinalizer(obj, objectDeleted)
	objectsLock.Unlock()
}

func Untrack(obj interface{}) {
	ptr := fmt.Sprintf("%p", obj)
	trackedObj, ok := objects[ptr]
	if !ok {
		panic(fmt.Sprintf("Not tracking object: %+v", obj))
	}
	trackedObj.tracked = false
}

func objectDeleted(obj interface{}) {
	objectsLock.Lock()
	delete(objects, fmt.Sprintf("%p", obj))
	objectsLock.Unlock()
}

func LogTracked() {
	objectsLock.Lock()
	defer objectsLock.Unlock()

	for _, obj := range objects {
		glog.Infof("Tracked %q since %v", obj.name, obj.timestamp)
	}
}


type byTimestamp []*trackedObject

func (a byTimestamp) Len() int           { return len(a) }
func (a byTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTimestamp) Less(i, j int) bool {
	/*if a[i].tracked != a[j].tracked {
		return a[i].tracked
	}*/
	return a[i].timestamp.Before(a[j].timestamp)
}

func outputTracked(w http.ResponseWriter, req *http.Request) {
	// Organize by name.
	objectsLock.Lock()
	perName := make(map[string]map[string][]*trackedObject, len(objects))
	for _, obj := range objects {
		c, ok := perName[obj.name]
		if !ok {
			c = make(map[string][]*trackedObject)
			perName[obj.name] = c
		}
		c[obj.label] = append(c[obj.label], obj)
	}
	objectsLock.Unlock()

	for name, objs := range perName {
		w.Write([]byte(fmt.Sprintf("Tracked %q has %d labels\n", name, len(objs))))
		for label, vals := range objs {
			sort.Sort(byTimestamp(vals))
			numExp := 0
			for i := range vals {
				if !vals[i].tracked {
					numExp++
				}
			}
			w.Write([]byte(fmt.Sprintf("- %q with %d objects and %d untracked (oldest %v)\n", label, len(vals), numExp, time.Since(vals[0].timestamp))))
		}
	}
}

func StartTracking() {
	http.HandleFunc("/tracked", outputTracked)
	http.HandleFunc("/gc", func(w http.ResponseWriter, req *http.Request) {
		runtime.GC()
	})

	c := time.Tick(5 * time.Minute)
	for _ = range c {
		glog.Infof("Printing tracked...")
		LogTracked()
	}
}
