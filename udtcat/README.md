# udtcat - netcat with go-udtwrapper


## Building

First, install the project:

```
go get github.com/jbenet/go-udtwrapper/udtcat
cd github.com/jbenet/go-udtwrapper/udtcat
```

But first! You'll need to build + install the C++ UDT dylib:

```sh
cd ../udt4

# see the README and compile according to its params
make -e os=XXX arch=YYY

# add to dyld path
export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$(pwd)
```

Then, build udtcat:

```
cd ../udtcat
go install
```

Optional:

```
# add udtcat dir to your path so you can f

```

## Usage

```
udtcat - UDT netcat in Go

Usage:

  listen: ./udtcat [<local address>] <remote address>
  dial:   ./udtcat -l <local address>

Address format is Go's: [host]:port
  -l=false: listen for connections (short)
  -listen=false: listen for connections
  -v=false: verbose debugging
```

## Example

```
# in one terminal, listen at :1111
./udtcat -l :1111

# in another, connect to it
./udtcat 127.0.0.1:1111
```

And stdio <3 means you can move files across:

```
# in one terminal, listen at :1111
./udtcat -l :1111 <INFILE >OUTFILE

# in another, connect to it
./udtcat 127.0.0.1:1111 <INFILE >OUTFILE
```
