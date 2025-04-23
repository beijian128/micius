package util

import (
	_ "expvar" // 监控变量
	"fmt"
	"net/http"
	_ "net/http/pprof" // pprof设定http关联
	"os"
	"runtime"
	"runtime/pprof"
)

// HTTPPProf ...
// go tool pprof http://localhost:12345/debug/pprof/heap
// go tool pprof http://localhost:12345/debug/pprof/profile
// go tool pprof http://localhost:12345/debug/pprof/block
// wget http://localhost:12345/debug/pprof/trace?seconds=5
// view main page http://localhost:12345/debug/pprof
func HTTPPProf(host string, port int) {
	if port < 1024 || port > 65535 {
		fmt.Println("pprof not start, illegal port:", port)
		return
	}
	go func() {
		addr := fmt.Sprintf("%s:%d", host, port)
		fmt.Println("pprof listen at:", addr, ", err:", http.ListenAndServe(addr, nil))
	}()
}

// StartCPUProfile cpuprofile=filename
func StartCPUProfile(cpuprofile string) {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			fmt.Fprint(os.Stderr, "StartCPUProfile", err)
		}
		pprof.StartCPUProfile(f)
	}
}

// StopCPUProfile ...
// go tool pprof exename proffilename
func StopCPUProfile() {
	pprof.StopCPUProfile()
}

// StartMemProfile default value = 512 * 1024
// go tool pprof exename proffilename
func StartMemProfile(memprofilerate int) {
	if memprofilerate > 0 {
		runtime.MemProfileRate = memprofilerate
	}
}

// StopMemProfile filename
func StopMemProfile(memprofile string) {
	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can not create mem profile output file: %s", err)
			return
		}
		if err = pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "Can not write %s: %s", memprofile, err)
		}
		f.Close()
	}
}

// StartBlockProfile default value = 1 prfile every thing
// go tool pprof exename proffilename
func StartBlockProfile(blockprofilerate int) {
	if blockprofilerate > 0 {
		runtime.SetBlockProfileRate(blockprofilerate)
	}
}

// StopBlockProfile file name
func StopBlockProfile(blockprofile string) {
	if blockprofile != "" {
		f, err := os.Create(blockprofile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can not create block profile output file: %s", err)
			return
		}
		if err = pprof.Lookup("block").WriteTo(f, 0); err != nil {
			fmt.Fprintf(os.Stderr, "Can not write %s: %s", blockprofile, err)
		}
		f.Close()
	}
}
