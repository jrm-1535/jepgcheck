
package main

import (
    "fmt"
    "flag"
    "math/bits"
    "io"
    "os"
    "strings"
    "strconv"
    "github.com/jrm-1535/jpeg"
)

const (
    VERSION     = "0.3"
    BEGIN       = 0
    END         = (1<<bits.UintSize)-1

    HELP        = 
`jcheck [-h] [-v] [-oh=<class>]
        [-w] [-rp] [-m] [-mcu] [-du] [-b=nn] [-e=pp]
        [-t] [-meta=<a>[:<s>] [-qu=<d>s|x|b] [-en=<c>:<d>[:f]s|x|b] [-sc=<n>[:f]s|x|b]
        [-tidyup] [-rmeta=<a>:<s>] [-sthumb=<i>:<path>] [-o=name] filepath

    Check if a file is a valid jpeg document, allowing to print internal
    information about the jpeg encoding, to show errors during analysis, to fix
    a few minor errors in the jpeg file format, to save embedded thumbnails as
    separate JPEG files and to save the raw decoded RGB data.

    General options:

        -h                      print this short help message and exit
        -v                      print current jcheck version and exit
        -oh=<class>             print longer <class> options help and exit
                                <class> can be: parse, display, modify or save

    Parsing options:                    for more details -oh=parse

        -w                      warn about issues during parsing
        -rp                     recursively parse embedded jpeg pictures
        -m                      print markers as parsing goes
        -mcu                    print detailed mcu parsing (very verbose)
        -du                     print data units from mcu (extremely verbose)
        -b=<nn>                 begin printing mcu/du at mcu #nn (default 0)
        -e=<pp>                 end printing at mcu #pp (default end of scan)

    Display options:                    for more details -oh=display

        -t                      print jpeg tables in file order.
        -meta=<a>[:<s>]*        print metadata from app segment(s).
        -qu=<d>s|x|b            print quantization matrixes
        -en=<c>:<d>[:<f>]s|x|b  print entropy tables.
        -sc=<n>[:f]s|x|b        print scan information

    Modification options:               for more details -oh=modify

        -tidyup                 fix common errors and clean file during analysis
        -rmeta=<a>[:<s>]        remove non-critical metadata from the file.

    Saving options:                     for more details -oh=save

        -sthumb=<t>:<p>         save embedded thumbnail into new file
        -spict=[<o>[,<f>]:]<p>  Save main picture as raw RGB samples
        -o name                 output the modified JPEG data to a new file

    filepath is the path to the file to process

`
    PARSE_OPTIONS =
`
    Parsing options:

        -w          warn about inconsistencies and errors during parsing
        -rp         recursively parse all embedded jpeg pictures (thumbnails).
        -m          print markers and offsets as parsing goes
        -mcu        print detailed mcu parsing (very verbose)
        -du         print each data unit extracted from mcu (extremely verbose)
        -b=<nn>     begin printing mcu and/or du at mcu #nn (default 0)
        -e=<pp>     end printing mcu/du at mcu #pp (default end of scan)

`

    DISPLAY_OPTIONS = 
`
    Display options:

         -t         print all jpeg tables after parsing, including all
                    quantization and entropy tables, in file order.
        -meta=<appId>[:<sid>]*[,<appId>[:<sid>]*
                    print metadata from app segments. The argument is the list
                    of app segments identified by their index n (0 for app0 to
                    15 for app15), or the special value -1 to print metadata
                    from all app segments, optionally followed by a list of
                    subset ids. This is intended for app segments that include
                    several containers, such as the app1 used with TIFF ifds.
                    The following standard sids can be used: 0 for main (TIFF)
                    ifd, 1 for thumbnail ifd, 2 for exif ifd, 3 for gps ifd, 4
                    for interoperability ifd, 5 for maker note ifd and 6 for a
                    possible maker-note embedded ifd.
                    For example, -meta=0,1:0:2 will show all metadata available
                    in app0 and only ifds 0 and 2 in app1 (exif) segment.
        -qu=<d>s|x|b[,<d>s|x|b]*
                    print quantization matrixes
                    d is the table destination from 0 to 4, or * for all
                    destinations. The following letter, s, x or b requests
                    respectively that a standard form, an extra version or
                    both standard and extra version be used (default to
                    standard if absent).
                    In case of quantization, the standard form is the list of
                    coefficients in zigzag order, whereas the extra from is the
                    quantization matrix ordered by rows.
        -en=<c>:<d>[:<f>]s|x|b*[,<c>:<d>[:<f>*]s|x|b]*
                    print entropy tables.
                    c is the table class, DC or AC or *, d is the table
                    destination, from 0 to 4, or * for all destinations within
                    the class and f is an optional frame number (0 by default).
                    The following letter, s, x or b requests respectively that
                    a standard form, an extra version or both standard and
                    extra version be used (default to standard if absent).
                    For example, -e=DC:1x,AC:* will show the DC table #1 in the
                    extra form and all AC tables for frame 0 in the standard
                    form, whereas -hs=*:*:1 will show 8 tables (4 destinations
                    for DC and for AC classes) for frame 1, in standard form.
                    In case of Huffman coding the standard form is the list od
                    code lengths and corresponding symbols, whereas the extra
                    form if the complete list of Huffman codes and corresponding
                    symbols sorted by increasing code length.

`

    MODIFY_OPTIONS = 
`
    Modification options:

        -tidyup     fix common errors and clean file during analysis
        -rmeta=<id>[:<sid>]*[,<id>[:<sid>]]*
                    remove non-critical metadata information from the file.
                    id is the jpeg app segment id (0 to 15, for app0 to app15)
                    or the special value -1 which means removing all existing
                    app segments, followed optionally by a list of sub ids if
                    the app segment contains multiple containers that can be
                    identified by ids. This is intended for app1 containing
                    multiple tiff ifd. In this case the list of sids identifies
                    all ifd ids to remove (whole ifds, it is not possible to
                    remove only a few specific exif tags from an ifd). See
                    option -meta for the how sid are used. If sids are absent,
                    the whole app segment is removed.
                    For example, -r=1,13 will remove whole segments APP1 and
                    APP13, whereas -r=0,1:5:6 will remove the whole APP0 segment
                    and keep most of the APP1 (tiff/exif) ifds, removing only
                    the maker note (5) and the embedded preview picture (6).

`

    SAVE_OPTIONS =
`
    Saving options
        -sthumb=<tid>:<path>[,<tid>:<path>]
                    save thumbnail identified by id. A JFIF file may include a
                    single thumbnail whereas an EXIF file may include both a
                    thumbnail image and a preview image within the app1 segment.
                    Each thumbnail image is stored in a new file at their given
                    path. By convention, tid=0 refers always the main thumbnail
                    and tid=1 refers to a possible additional preview image.
        -spict=[<orientation>[,<format>]:]<path>
                    save the main picture possibly after transformation required
                    by <orientation> in the requested <format>.
                    <orientation> is similar to the tiff/exif orientation tag.
                    It is optional and if missing the tiff/exif value is used
                    if available, otherwise the default picture orientation is
                    used. <orientation> can be given as:
                    TL (top side row 0, left side col 0: default)
                    TR (top side row 0, right side col 0: vertical mirror)
                    BR (bottom side row 0, right side col 0: 180 degree
                       clockwise rotation)
                    BL (bottom side row 0, left side col 0: horizontal mirror)
                    LT (left side row 0, top side col 0: horizontal mirror and
                        90 degree clockwise rotation)
                    RT (right side row 0, top side col 0: 90 degree clockwise
                        rotation)
                    RB (right side row 0, bottom side col 0: vertical mirror
                        and 90 degree clockwise rotation)
                    LB (left side row 0, bottom side col 0: 270 degree
                        clockwise rotation)
                    <format> indicates whether the picture should be stored as
                    color (RGB) or as black and white (Y). It is optional and
                    if missing it is assumed to mean using all available color
                    components. <format> can be given as either BW or RGB.  
                    Therefore, if BW is not specified and all Y, Cb, Cr are
                    available, the picture is stored as packed RGB (3 bytes per
                    pixel), otherwise it is stored as 1 byte (Y) per pixel.
                    Note that if <format> is given, a leading comma ',' is
                    required even if <orientation> is missing.
        -o  name    output the modified JPEG data to a new file
                    this option is meaningful if -rmeta and/or -tydyip were
                    specified (if nothing was modified, the files will be
                    similar if not identical).

`
)

type scTable struct {
    index, frame    int
    mode            jpeg.FormatMode
}

type quTable struct {
    dest, frame     int
    mode            jpeg.FormatMode
}

type enTable struct {
    class, dest,            // -1 if all, otherwise resp [0-1] and [0-3]
    frame           int     // -1 if all, otherwise [0-n]
    mode            jpeg.FormatMode
}

type metaIds struct {
    appId           int
    sIds            []int
}

type storeParameters struct {
    row0        jpeg.VisualSide
    col0        jpeg.VisualSide
    bw          bool
    path        string
}

type jpgArgs struct {
    input, output   string
    control         jpeg.Control
    tables          bool
    meta            []metaIds
    quTables        []quTable
    enTables        []enTable
    scTables        []scTable
    rmActions       []metaIds
    svActions       []jpeg.ThumbSpec
    sPicture        storeParameters
}

var format = [...]string { "BW", "RGB" }
func getFormat( f string ) (bool, error) {
    for i, fs := range format {
        if f == fs {
            return i == 0, nil
        }
    }
    return false, fmt.Errorf("format %s is not recognized\n", f )
}

var orientation = [...]string { "TL", "TR", "BR", "BL", "LT", "RT", "RB", "LB" }
func getOrientation( o string ) (r, c jpeg.VisualSide, err error) {
    for i, os := range orientation {
        if o == os {
            switch i {
            case 0: r = jpeg.Top; c = jpeg.Left
            case 1: r = jpeg.Top; c = jpeg.Right
            case 2: r = jpeg.Bottom; c = jpeg.Right
            case 3: r = jpeg.Bottom; c = jpeg.Left
            case 4: r = jpeg.Left; c = jpeg.Top
            case 5: r = jpeg.Right; c = jpeg.Top
            case 6: r = jpeg.Right; c = jpeg.Bottom
            case 7: r = jpeg.Left; c = jpeg.Bottom
            }
            return
        }
    }
    err = fmt.Errorf("orientation %s is not recognized\n", o )
    return
}

// undefined orientation is indicated by row0 and col0 both zero
func parseSpict( spict string ) ( res storeParameters, err error ) {
    parts := strings.Split( spict, ":" )
    if len(parts) > 2 {
        return res, fmt.Errorf("Save picture: syntax error: too many ':' in %s\n",
                                spict )
    }
    if len(parts) == 2 {
        spict = parts[1]
        part := parts[0]
        params := strings.Split( part, "," )
        if len(params) > 2 {
            return res, fmt.Errorf("Save picture: syntax error: too many ',' in %s\n",
                                    part )
        }
        if len(params) == 2 {
            res.bw, err = getFormat( params[1] )
            if err != nil {
                return res, fmt.Errorf("Save picture: syntax error: %v\n", err)
            }
        }
        if params[0] != "" {
            res.row0, res.col0, err = getOrientation( params[0] )
            if err != nil {
                return res, fmt.Errorf("Save picture: syntax error: %v\n", err)
            }
        }
    }
    res.path = spict
    return
}

func parseSthumb( sthumb string ) (res []jpeg.ThumbSpec, err error) {
    // -sthumb=<tid>:<path>[,<tid>:<path>]
    parts := strings.Split( sthumb, "," )
    for _, part := range parts {
        specs := strings.Split( part, ":" )
        if len(specs) != 2 {
            return nil, fmt.Errorf("Save Thumbnails: missing path or id: %s\n",
                                   part )
        }

        v, err := strconv.ParseInt(specs[0], 0, 64);
        if err != nil || v < 0 || v > 1 {
            return nil, fmt.Errorf( "invalid Id: %s\n", specs[0] )
        }
        res = append( res, jpeg.ThumbSpec{ specs[1], int(v) } )
    }
    return
}

func parseMeta( rem string, remove bool ) (res []metaIds, err error ) {
// -meta=<appId>[:<sid>]*[,<appId>[:<sid>]]*
// -rmeta=<appId>[:<sid>]*[,<appId>[:<sid>]]*
    var lowBound int64
    if remove {
        lowBound = 1
    } else {
        lowBound = 0
    }
    parts := strings.Split( rem, "," )
    for _, part := range parts {
        specs := strings.Split( part, ":" )
        v, e := strconv.ParseInt(specs[0], 0, 64);
        if e != nil || (( v < lowBound || v > 15 ) && v != -1) {
            err = fmt.Errorf( "invalid Id: %s\n", specs[0] )
            return
        }
        id := int(v)
        if len(specs) == 1 || id == -1 {
            res = append( res, metaIds{ id, []int{} } )
            return
        }
        var sids []int
        for _, sid := range specs[1:] {  // id positive integer

            v, err := strconv.ParseInt(sid, 0, 64); if err != nil || v < lowBound {
                return nil, fmt.Errorf( "invalid Id: %s\n", sid )
            }
            id := int(v)
            sids = append( sids, id )
        }
        res = append( res, metaIds{ id, sids } )
    }
    return res, nil
}

func getModePart( p string ) (jpeg.FormatMode, string, error) {
    var m jpeg.FormatMode
    if len(p) < 1 {
        return 0, p, fmt.Errorf( "syntax error" )
    }
    switch p[len(p)-1] {
    case 's':
        m = jpeg.Standard
        p = p[:len(p)-1]
    case 'x':
        m = jpeg.Extra
        p = p[:len(p)-1]
    case 'b':
        m = jpeg.Both
        p = p[:len(p)-1]
    default:
        m = jpeg.Standard
    }
    return m, p, nil
}

func parseScan( scan string ) (res []scTable, err error) {
    parts := strings.Split( scan, "," )
    for _, part := range parts {
        var mode jpeg.FormatMode
        mode, part, err = getModePart( part ); if err != nil {
            err = fmt.Errorf( "Scan table %v: -sc=%s\n", err, scan )
            return
        }
        specs := strings.Split( part, ":" )
        if len(specs) > 2 {
            return nil, fmt.Errorf( "Scan table syntax error: -sc=%s\n", scan )
        }
        var index, frame int
        if specs[0] == "*" {
            index = -1
        } else {
            v, err := strconv.ParseInt(specs[0], 0, 64); if err != nil || v < 0 || v > 3 {
                return nil, fmt.Errorf(
                    "invalid Scan table index: %s\n", specs[0] )
            }
            index = int(v)
        }
        if len(specs) > 1 {
            if specs[1] == "*" {
                frame = -1
            } else {
                v, err := strconv.ParseInt(specs[1], 0, 64);  if err != nil || v < 0 {
                    return nil, fmt.Errorf( "invalid Scan table frame: %s\n", specs[1] )
                }
                frame = int(v)
            }
        }
        res = append( res, scTable{ index, frame, mode } )
    }
    return res, nil
}

func parseQuantization( quantization string ) (res []quTable, err error) {
    parts := strings.Split( quantization, "," )
    for _, part := range parts {
        var mode jpeg.FormatMode
        mode, part, err = getModePart( part ); if err != nil {
            err = fmt.Errorf( "Quantization table %v: -qu=%s\n", err, quantization )
            return
        }
        specs := strings.Split( part, ":" )
        if len(specs) > 2 {
            return nil, fmt.Errorf( "Quantization table syntax error: -qu=%s\n",
                                    quantization )
        }
        var dest, frame int
        if specs[0] == "*" {
            dest = -1
        } else {
            v, err := strconv.ParseInt(specs[0], 0, 64); if err != nil || v < 0 || v > 3 {
                return nil, fmt.Errorf(
                    "invalid Quantization table destination: %s\n", specs[0] )
            }
            dest = int(v)
        }
        if len(specs) > 1 {
            if specs[1] == "*" {
                frame = -1
            } else {
                v, err := strconv.ParseInt(specs[1], 0, 64);  if err != nil || v < 0 {
                    return nil, fmt.Errorf( "invalid Quantization table frame: %s\n", specs[1] )
                }
                frame = int(v)
            }
        }
        res = append( res, quTable{ dest, frame, mode } )
    }
    return res, nil
}

func parseEntropy( entropy string ) (res []enTable, err error) {

    parts := strings.Split( entropy, "," )
    for _, part := range parts {
        var mode jpeg.FormatMode
        mode, part, err = getModePart( part ); if err != nil {
            return nil, fmt.Errorf( "Entropy table %v: -en=%s\n", err, entropy )
        }
        specs := strings.Split( part, ":" )
        if len(specs) < 2 || len(specs) > 3 {
            return nil, fmt.Errorf( "Entropy table syntax error: -en=%s\n", entropy )
        }
        var class, dest, frame int
        switch specs[0] {
        case "*":
            class = -1
        case "DC":
            class = 0
        case "AC":
            class = 1
        default:
            return nil, fmt.Errorf( "invalid Entropy table class: %s\n", specs[0] )
        }
        if specs[1] == "*" {
            if class != -1 {
                return nil, fmt.Errorf(
                     "Unsupported case: specific destination for all classes\n" )
            }
            dest = -1
        } else {
            if class == -1 {
                return nil, fmt.Errorf(
                     "Unsupported case: all destinations for specific class\n" )
            }
            v, err := strconv.ParseInt(specs[1], 0, 64); if err != nil || v < 0 || v > 3 {
                return nil, fmt.Errorf( "invalid Entropy table destination: %s\n", specs[1] )
            }
            dest = int(v)
        }

        if len(specs) > 2 {
            if specs[2] == "*" {
                frame = -1
            } else {
                v, err := strconv.ParseInt(specs[2], 0, 64);  if err != nil || v < 0 {
                    return nil, fmt.Errorf( "invalid Entropy table frame: %s\n", specs[2] )
                }
                frame = int(v)
            }
        }
        res = append( res, enTable{ class, dest, frame, mode } )
    }
    return res, nil
}

var classes = [...]string{ "parse", "display", "modify", "save" }
var help    = [...]string{ PARSE_OPTIONS, DISPLAY_OPTIONS, MODIFY_OPTIONS, SAVE_OPTIONS }
func optionHelp( c string ) {
    for i := 0; i < len(classes); i++ {
        if classes[i] == c {
            fmt.Fprintf( flag.CommandLine.Output(), help[i] )
            os.Exit(0)                
        }
    }
    fmt.Printf( "Unknown option class: %s\n", c )
    os.Exit(2)
}

func getArgs( ) (* jpgArgs, error) {

    pArgs := new( jpgArgs )

    var version bool
    flag.BoolVar( &version, "v", false, "print jcheck version and exits" )
    flag.BoolVar( &pArgs.control.Markers, "m", false, "print markers and offsets as parsing goes" )
    flag.BoolVar( &pArgs.control.Warn, "w", false, "warn of errors during parsing" )
    flag.BoolVar( &pArgs.control.Mcu, "mcu", false, "print minimum coded unit processing" )
    flag.BoolVar( &pArgs.control.Du, "du", false, "print resulting data unit" )
    flag.UintVar( &pArgs.control.Begin, "b", BEGIN, "begin printing mcu/du at mcu #nn (default 0)" )
    flag.UintVar( &pArgs.control.End, "e", END, "end printing mcu/du at mcu #pp (default end of scan)" )
    flag.BoolVar( &pArgs.control.Recurse, "rp", false, "Recursively parse embedded JPEG pictures" )
    flag.BoolVar( &pArgs.control.TidyUp, "tidyup", false, "try fixing errors during analysis" )

    flag.BoolVar( &pArgs.tables, "t", false, "print jpeg tables during analysis" )
    var meta string
    flag.StringVar( &meta, "meta", "", "print metadata" )
    var quantizer string
    flag.StringVar( &quantizer, "qu", "", "print quantizer matrixes" )
    var entropy string
    flag.StringVar( &entropy, "en", "", "print entropy tables" )
    var scan string
    flag.StringVar( &scan, "sc", "", "print scan tables" )
    var remove string
    flag.StringVar( &remove, "rmeta", "", "remove metadata" )
    var sthumb string
    flag.StringVar( &sthumb, "sthumb", "", "save embedded thumbnail in a new file" )
    var spict string
    flag.StringVar( &spict, "spict", "", "save decompressed picture in a new file" )
    flag.StringVar( &pArgs.output, "o", "", "output modified JPEG data to the file`name`" )
    var soptions string
    flag.StringVar( &soptions, "oh", "", "detailed options help" )

    flag.Usage = func() {
        fmt.Fprintf( flag.CommandLine.Output(), HELP )
    }
    flag.Parse()
    if version {
        fmt.Fprintf( flag.CommandLine.Output(), "pdfCheck version %s\n", VERSION )
        os.Exit(0)
    }
    if soptions != "" {
        optionHelp( soptions )
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
    if meta != "" {
        mids, err := parseMeta( meta, false )
        if err != nil {
            return nil, fmt.Errorf( "getArgs: %w", err )
        }
        pArgs.meta = mids
    }
    if entropy != "" {
        enTables, err := parseEntropy( entropy )
        if err != nil {
            return nil, fmt.Errorf( "getArgs: %w", err )
        }
        pArgs.enTables = enTables
    }
    if quantizer != "" {
        quTables, err := parseQuantization( quantizer )
        if err != nil {
            return nil, fmt.Errorf( "getArgs: %w", err )
        }
        pArgs.quTables = quTables
    }
    if scan != "" {
        scTables, err := parseScan( scan )
        if err != nil {
            return nil, fmt.Errorf( "getArgs: %w", err )
        }
        pArgs.scTables = scTables
    }
    if remove != "" {
        rmActions, err := parseMeta( remove, true )
        if err != nil {
            return nil, fmt.Errorf( "getArgs: %w", err )
        }
// Debug
        for _, ra := range rmActions {
            fmt.Printf( "app%d ", ra.appId)
            for _, id := range ra.sIds {
                fmt.Printf( ":%d ", id )
            }
            fmt.Printf( "\n" )
        }
// end debug
        pArgs.rmActions = rmActions
    }
    if sthumb != "" {
        svActions, err := parseSthumb( sthumb )
        if err != nil {
            return nil, fmt.Errorf( "getArgs: %w", err )
        }
// Debug
        for _, xa := range svActions {
            fmt.Printf( "Save thumbnail %d:%s\n", xa.ThId, xa.Path )
        }
// end debug
        pArgs.svActions = svActions
    }

    if spict != "" {
        sparams, err := parseSpict( spict )
        if err != nil {
            return nil, fmt.Errorf( "getArgs: %w", err )
        }
// Debug
        fmt.Printf( "Save picture: orientation row0=%v col0=%v BW=%v to path %s\n",
                    sparams.row0, sparams.col0, sparams.bw, sparams.path )
// end debug
        pArgs.sPicture = sparams
    }

    if pArgs.output == "" {
        if pArgs.control.TidyUp {
            fmt.Printf( "Warning: although tydying up the original file " +
                        "is requested, NO output file is requested\n" )
            fmt.Printf( "         proceeding anyway\n" )
        }
        if len(pArgs.rmActions) != 0 {
            fmt.Printf( "Warning: although removing metatata from the original"+
                        " file is requested, NO output file is requested\n" )
            fmt.Printf( "         proceeding anyway\n" )
        }
    } else {
        if ! pArgs.control.TidyUp && len(pArgs.rmActions) == 0 {
            fmt.Printf( "Warning: although an output file is requested, " +
                        "tidying up or removing metadata from the original " +
                        "file is NOT requested\n" )
            fmt.Printf( "         proceeding anyway\n" )
        }
    }
    pArgs.input = arguments[0]
    return pArgs, nil
}

func processMeta( w io.Writer, jpg *jpeg.Desc, args *jpgArgs ) (err error) {
    for _, mid := range args.meta {
        _, err = jpg.FormatMetadata( w, mid.appId, mid.sIds )
        if err != nil {
            break
        }
    }
    return
}

func processTables( w io.Writer, jpg *jpeg.Desc, args *jpgArgs ) error {
    if args.tables {
        n, err := jpg.FormatSegments( w )
        if err == nil {
            fmt.Printf( "jpegcheck: formatted %d bytes\n", n )
        }
        return err
    }
    return nil
}

func processQuantization( w io.Writer, jpg *jpeg.Desc, args *jpgArgs ) (err error) {

tableLoop:
    for _, qt := range args.quTables {
        if qt.frame == -1 {
            nFrames := jpg.GetNumberOfFrames()
            for i := uint(0); i < nFrames; i++ {
                _, err = jpg.FormatEncodingTable(
                                os.Stdout, i, jpeg.Quantization, qt.dest, qt.mode )
                if err != nil {
                    break tableLoop
                }
            }
        } else {
            _, err = jpg.FormatEncodingTable(
                      os.Stdout, uint(qt.frame), jpeg.Quantization, qt.dest, qt.mode )
            if err != nil {
                break tableLoop
            }
        }
    }
    return
}

func processEntropy( w io.Writer, jpg *jpeg.Desc, args *jpgArgs ) (err error) {

tableLoop:
    for _, et := range args.enTables {
        if et.class == -1 {
            if et.frame == -1 {
                nFrames := jpg.GetNumberOfFrames()
                for i := uint(0); i < nFrames; i++ {
                    _, err = jpg.FormatEncodingTable(
                                    os.Stdout, i, jpeg.Entropy, -1, et.mode )
                    if err != nil {
                        break tableLoop
                    }
                }
            } else {
                _, err = jpg.FormatEncodingTable(
                          os.Stdout, uint(et.frame), jpeg.Entropy, -1, et.mode )
                if err != nil {
                    break tableLoop
                }
            }
        } else { // class DC [0-3], class AC [4-7]
            dest := et.class * 4 + et.dest
            if et.frame == -1 {
                nFrames := jpg.GetNumberOfFrames()
                for i := uint(0); i < nFrames; i++ {
                    _, err = jpg.FormatEncodingTable(
                                    os.Stdout, i, jpeg.Entropy, dest, et.mode )
                    if err != nil {
                        break tableLoop
                    }
                }
            } else {
                _, err = jpg.FormatEncodingTable(
                        os.Stdout, uint(et.frame), jpeg.Entropy, dest, et.mode )
                if err != nil {
                    break tableLoop
                }
            }
        }
    }
    return
}

func processScan(w io.Writer, jpg *jpeg.Desc, args *jpgArgs ) (err error) {

tableLoop:
    for _, sc := range args.scTables {
        if sc.frame == -1 {
            nFrames := jpg.GetNumberOfFrames()
            for i := uint(0); i < nFrames; i++ {
                _, err = jpg.FormatEncodingTable(
                                os.Stdout, i, jpeg.Scan, sc.index, sc.mode )
                if err != nil {
                    break tableLoop
                }
            }
        } else {
            _, err = jpg.FormatEncodingTable(
                      os.Stdout, uint(sc.frame), jpeg.Scan, sc.index, sc.mode )
            if err != nil {
                break tableLoop
            }
        }
    }
    return
}

func processSave( jpg *jpeg.Desc, args *jpgArgs ) (err error) {
    if len(args.svActions) > 0 {
        err = jpg.SaveThumbnail( args.svActions )
    }
    return
}

func processRemove( jpg *jpeg.Desc, args *jpgArgs ) (err error) {

    for _, rm := range args.rmActions {
        err = jpg.RemoveMetadata( rm.appId, rm.sIds )
        if err != nil {
            break;
        }
    }
    return
}

func main() {

    process, err := getArgs()
    if err != nil {
        fmt.Printf( "jpegcheck: %v", err )
        return
    }

    fmt.Printf( "jpegcheck: checking file %s\n", process.input )

    jpg, err := jpeg.Read( process.input, &process.control )
    if err != nil {
        fmt.Printf( "%v\n", err )
    }
    jpg.FormatImageInfo( os.Stdout )
/*
    jpg.FormatFrameInfo( os.Stdout, 0 )
    jpg.FormatEncodingTable( os.Stdout, 0, jpeg.Quantization, -1 )
    jpg.FormatEncodingTable( os.Stdout, 0, jpeg.Entropy, -1 )
*/
    if jpg != nil && jpg.IsComplete( ) {

        jpg.FormatFrameInfo( os.Stdout, 0 )
        err = processTables( os.Stdout, jpg, process )
        if err != nil {
            fmt.Printf( "jpegcheck: %v", err )
            return
        }
        err = processMeta( os.Stdout, jpg, process )
        if err != nil {
            fmt.Printf( "jpegcheck: %v", err )
            return
        }
        err = processQuantization( os.Stdout, jpg, process )
        if err != nil {
            fmt.Printf( "jpegcheck: %v", err )
            return
        }
        err = processEntropy( os.Stdout, jpg, process )
        if err != nil {
            fmt.Printf( "jpegcheck: %v", err )
            return
        }
        err = processScan( os.Stdout, jpg, process )
        if err != nil {
            fmt.Printf( "jpegcheck: %v", err )
            return
        }

        err = processSave( jpg, process )
        if err != nil {
            fmt.Printf( "jpegcheck: %v", err )
            return
        }
        err = processRemove( jpg, process )
        if err != nil {
            fmt.Printf( "jpegcheck: %v", err )
            return
        }

        actualL, dataL := jpg.GetActualLengths()
        fmt.Printf( "Actual JPEG length: %d (original data length: %d)\n", actualL, dataL )

        if process.output != "" {
            fmt.Printf( "Generating a copy as '%s'\n", process.output )
            var n int
            n, err = jpg.Write( process.output )
            if err != nil {
                fmt.Printf( "jpegcheck: %v", err )
                return
            } else {
                fmt.Printf( "jpegcheck: written %d bytes\n", n )
            }
        }
        // FIXME
        if err == nil {
            _, err = jpg.FormatFrameComponent( os.Stdout, 0, 0 )
            if err != nil {
                fmt.Printf( "jpegcheck: %v", err )
                return
            }
            _, err = jpg.FormatFrameComponent( os.Stdout, 0, 1 )
            if err != nil {
                fmt.Printf( "jpegcheck: %v", err )
                return
            }
            _, err = jpg.FormatFrameComponent( os.Stdout, 0, 2 )
            if err != nil {
                fmt.Printf( "jpegcheck: %v", err )
                return
            }
        }
/*
        _, err = jpg.MakeFrameRawPicture( 0 )
        if err != nil {
            fmt.Printf( "jpegcheck: %v", err )
            return
        }
*/
        if process.sPicture.path != "" {
            var orientation *jpeg.Orientation
            if process.sPicture.row0 == 0 && process.sPicture.col0 == 0 {
                orientation, err = jpg.GetImageOrientation()
                if err != nil {
                    fmt.Printf( "jpegcheck: save picture: no tiff/exif orientation specified: %v", err )
                } else {
                    fmt.Printf( "jpegcheck: save picture using tiff/exif orientation:\n" )
                    side := []string { "Left", "Top", "Right", "Bottom" }
                    effect := []string {
                            "None", "VerticalMirror", "Rotate90",
                            "VerticalMirrorRotate90", "HorizontalMirror",
                            "Rotate180", "HorizontalMirrorRotate90", "Rotate270" }
                    fmt.Printf( "  Source app%d Row 0 at %s, Column 0 at %s (effect: %s)\n",
                                orientation.AppSource, 
                                side[orientation.Row0], side[orientation.Col0],
                                effect[orientation.Effect] )
                }
            } else {
                orientation = new(jpeg.Orientation)
                orientation.Row0 = process.sPicture.row0
                orientation.Col0 = process.sPicture.col0
            }
            var nc, nr uint
            var n int
            nc, nr, n, err = jpg.SaveRawPicture(process.sPicture.path,
                                                process.sPicture.bw, orientation)
            if err != nil {
                fmt.Printf( "jpegcheck: save picture: %v", err )
            } else {
                fmt.Printf( "Saved %s as nCols=%d nRows=%d size %d\n",
                            process.sPicture.path, nc, nr, n )
            }
        }
    }
}
