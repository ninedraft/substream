# substreamer

Stream music from filesystem as HTTP streams.

In heavy development.

## Build
```bash
go build -o substreamer.exe ./cmd/substreamer
```

## Run
```bash 
substreamer.exe -file <path to music file> -addr <address to listen on>
```

## Usage

- `/music` - music stream
- `/music/cover` - cover art if available
- `/` - status web page

## TODOS
- [ ] scan directory for music files instead of specifying file
- [ ] add support for other file types
- [ ] prettify status page
