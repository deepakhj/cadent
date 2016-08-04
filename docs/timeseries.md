

### TimeSeries

The core of things for writers/caching (not really used at all in the simply Constist Hashing or Statsd modes).

There are a number of ways for store things.  Some are very good a RAM compression and others are good for ease of use
compatibility, and other internal uses as explained below.

Some definitions:

    - DeltaOfDeltas:

        Since 99.9% of the time "Time" is moving foward there is not need to store the full "int64" each time
        So we store a
        "start" time (T0),
        the next value is then T1 - T0 =  DeltaT1,
        the next value is then T2 - T1 = DeltaT2

        To get a time a the "Nth" point we simply need to

        TN = T0 + Sum(DeltaI, I -> {1, N})

    - VarBit:

        A "Variable Bit encoding" which will store a value in the smallest acctual size it can be stored int

        If the type is an "int64" but the value is only 0 or 1, just store one byte (plus some bits to say what it was)
        if the value is 1000, then only store 2 bytes,
        etc.

        If the type is a float64, but the value is just an int of some kind it will use the int encodings above

    - "Smart"Encoding:

        You'll notice that we store 7 64 bit numbers in the StatMetric.  Sometimes (alot of times) we only have
        a Count==1 in the above which means that all the floats64 are the same (or they better be).  If that's the
        case, we only store one float value (the Sum) (and, depending on the format below, a bit that tells us this fact)

    - TimeResolution:

        Some series (gob, protobuf, gorilla) can use the "int32" for seconds (lowres) of time and
        not int64 (highres) for nanosecond resolution.
        Lowres is the "default" behavior as most of the time our flush times are in the SECONDS not less


#### GOB + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

The gob format which is a GoLang Specific encoding https://golang.org/pkg/encoding/gob/ that uses the VarBit encoding internally.

This method is pretty efficent.  But it is golang specific so it's not too portable.  But may be good for other uses
(like quick file appenders or something in the future).

#### ProtoBuf + VarBit + SmartEncoding + TimeResolution

Standard protobuf encoder github.com/golang/protobuf/proto using the https://github.com/gogo/protobuf generator

Since protobuf is pretty universal at this point (lots of lanuages can read it and use it) it's pretty portable
It's also a bit more efficent in space as the GOB form, due to the nicer encoding methods provided by gogo/protobuf

Also this is NOT time order sensitive, it simply stores each StatMetric "as it is" and it's simple an array
of these internally, so it's good for doing slice operations (i.e. walking time caches and SORTING by times)

This uses "gogofaster" for the generator of proto schemas

    go get github.com/gogo/protobuf/protoc-gen-gogofaster
    protoc --gogofaster_out=. *.proto


#### Json

The most portable format, but also the biggest (as we a storing strings in reality).  I'd only use this if you
need extream portability across things, as it's not really space efficent at all.

#### MsgPack

In terms of "space" consumed this is a little worse then gogofaster's ProtoBuf.
In terms of Decoding/Encoding speed it's on a level all its own.  Use this for things that favor speed vs space.

#### Gorilla + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

The most extream compression available.  As well as doing the same goodies mentioned, the core squeezer is the
Float64 compression it uses.  Read up on it here http://www.vldb.org/pvldb/vol8/p1816-teller.pdf.

This does NOT even remotely support out-of-time-order inputs.  The encoding interal to Cadent is a modified version
that allows for multi float encodings, Nano-second encodings and the "smart" encoding as well.

This is by far the best format to store things if your pieces can support it (both in ram and longterm), but due to
the forced timeordering, lack  of sorting.  It does not play well w/ many internal things.

The compression is also highly variable depending on incoming values, so it can be hard to "know" what storage or ram
constraints will be needed a-priori (unless you know the domain of your metrics well).


#### Repr

This is the "native" internal format for a metric cadent.  It's NOT very space conciderate (as it's vewry similare
to the json format, but w/ more stuff).  Basically don't use it for storage.


#### ZipGob  + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

Instead of a flat []byte buffer for gob encoding use the FLATE buffer (otherwise exactly the same as Gob).  While it
does some some space, due to the internals of the golang Flate, there is alot of GC churn and evils associated
with this one if used for a large number of series.  So this may be a good "final persist state" but it should get converted
out of this format for use elsewhere.

