// Copyright (C) 2015 Myfitnesspal

// testing out the pre-reg world with random stdin inputs
//
package main

import (
    "../../server/prereg"
    "fmt"
    "flag"
    "os"
    "bufio"
    "strings"

)

func main() {
    regConfigFile := flag.String("prereg", "", "File that contains the Regex/Filtering by key to various backends")
    flag.Parse()

    // deal with the pre-reg file
        lpr, err := prereg.ParseConfigFile(*regConfigFile)
        if err != nil {
            panic("Error parsing PreReg")
        }

    buf := bufio.NewReader(os.Stdin)
    for {
        line, err := buf.ReadString('\n')
        if err != nil {
            break
        }
        if len(line) == 0 {
            continue
        }
        line = strings.TrimSpace(line)

        flist := lpr.MatchingFilters(line)
        for idx, f := range flist {
            fmt.Printf(" Match #%2d: %s\n", idx, f.ToString())
        }
        if len(flist) ==0 {
            fmt.Printf("No Match\n")
        }

        // should at least give the default backend
        backs, _, _ := lpr.FirstMatchBackends(line)
        for idx, _ := range backs {
            fmt.Printf("Match #%2d: Backend: %s\n", idx, backs[idx])
        }

    }
}

