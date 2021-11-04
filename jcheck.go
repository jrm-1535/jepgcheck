
package main

import (
    "fmt"
    "flag"
    "math/bits"
    "os"
    "github.com/jrm-1535/jpeg"
)

const (
    VERSION     = "0.1"
    MARKERS     = false
    CONTENT     = false
    QUANTIZERS  = false
    LENGTHS     = false
    CODES       = false
    MCU         = false
    DU          = false
    BEGIN       = 0
    END         = (1<<bits.UintSize)-1
    FIX         = false

    HELP        = 
`jcheck [-h] [-v] [-m] [-t] [-q] [-hs] [-hc] [-mcu] [-du] [-f] [-b=nn] [-e=pp] [-o=name] filepath

    Check if a file is a valid jpeg document, allowing to print useful
    information about the jpeg encoding, show errors during analysis or
    fix some minor errors in the jpeg file format.

    Options:
        -h          print this help message and exits
        -v          print jcheck version and exits

        -m          print markers and offsets as parsing goes
        -t          print jpeg tables during analysis

        -q          print quantizer matrixes
        -hs         print huffman symbols by code length
        -hc         print array of huffman code/symbol

        -mcu        print detailed mcu processing (very verbose)
        -du         print each data unit extracted from mcu (extremely verbose)
        -b=nn       begin printing mcu/du at mcu #nn (default 0)
        -e=pp       end printing mcu/du at mcu #pp (default end of scan)

        -f          try fixing errors during analysing, instead of stopping

        -o  name    output the hopefully fixed data to a new file
                    this option is meaningful if -f is specified (if nothing was
                    fixed, the files will be similar if not identical).

    filepath is the path to the file to process

`
)

type jpgArgs struct {
    input, output   string
    control        jpeg.Control
}

func getArgs( ) (* jpgArgs ) {

    pArgs := new( jpgArgs )

    var version bool
    flag.BoolVar( &version, "v", false, "print jcheck version and exits" )
    flag.BoolVar( &pArgs.control.Markers, "m", MARKERS, "print markers and offsets as parsing goes" )
    flag.BoolVar( &pArgs.control.Content, "t", CONTENT, "print jpeg tables during analysis" )
    flag.BoolVar( &pArgs.control.Quantizers, "q", QUANTIZERS, "print quantizer matrixes" )
    flag.BoolVar( &pArgs.control.Lengths, "hs", LENGTHS, "print huffman symbols by code length" )
    flag.BoolVar( &pArgs.control.Codes, "hc", CODES, "print array of huffman code/symbol" )
    flag.BoolVar( &pArgs.control.Mcu, "mcu", MCU, "print minimum coded unit processing" )
    flag.BoolVar( &pArgs.control.Du, "du", DU, "print resulting data unit" )
    flag.UintVar( &pArgs.control.Begin, "b", BEGIN, "begin printing mcu/du at mcu #nn (default 0)" )
    flag.UintVar( &pArgs.control.End, "e", END, "end printing mcu/du at mcu #pp (default end of scan)" )
    flag.BoolVar( &pArgs.control.Fix, "f", FIX, "try fixing errors during analysis, instead of stopping" )
    //flag.BoolVar( &pArgs.warn, "w", WARN, "warn of errors during analysis" )
    flag.StringVar( &pArgs.output, "o", "", "output the hopefully fixed data to the file`name`" )

    flag.Usage = func() {
        fmt.Fprintf( flag.CommandLine.Output(), HELP )
    }
    flag.Parse()
    if version {
        fmt.Printf( "pdfCheck version %s\n", VERSION )
        os.Exit(0)
    }
    arguments := flag.Args()
    if len( arguments ) < 1 {
        fmt.Printf( "Missing the name of the file to process\n" )
        os.Exit(2)
    }
    if len( arguments ) > 1 {
        fmt.Printf( "Too many files specified (only 1 file at a time)\n" )
        os.Exit(2)
    }
    if pArgs.control.Fix {
        if pArgs.output == "" {
            fmt.Printf( "Warning: although fixing the original file is requested, NO output file is requested\n" )
            fmt.Printf( "         proceeding anyway\n" )
        }
    } else if pArgs.output != "" {
        fmt.Printf( "Warning: although an output file is requested, fixing the original file is NOT requested\n" )
        fmt.Printf( "         proceeding anyway\n" )
    }
    pArgs.input = arguments[0]
    return pArgs
}


func main() {

    process := getArgs()

    fmt.Printf( "jpegcheck: checking file %s\n", process.input )
    
    jpg, err := jpeg.ReadJpeg( process.input, &process.control )
    if err != nil {
        fmt.Printf( "%v\n", err )
    }
    if jpg != nil && jpg.IsComplete( ) {
        actualL, dataL := jpg.GetActualLengths()
        fmt.Printf( "Actual JPEG length: %d (original data length: %d)\n", actualL, dataL )

        if process.output != "" {
            fmt.Printf( "Generating a copy as '%s'\n", process.output )
            err = jpg.Write( process.output )
            if err != nil {
                fmt.Printf( "%v\n", err )
            }
        }
    }
}
