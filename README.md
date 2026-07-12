# O

code cli

url/key/model config in code

- read `AGENTS.md` and `SKILLS.md`
- multi-conversations
- read/write/edit/bash tool
- work only in project

## Benchmark

`read` uses a typed JSON argument structure and returns the file buffer without
copying it a second time. The benchmark retains the previous implementation as
`before`, so both paths run against the same fixture and executable:

```sh
go test -run '^$' -bench '^BenchmarkRead$' -benchmem -benchtime=5s
```

Representative result on Windows/amd64, Go 1.26.1, AMD Ryzen 7 8845HS:

| Size | Version | ns/op | MB/s | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: | ---: |
| 4 KiB | before | 880,772 | 4.65 | 14,554 | 85 |
| 4 KiB | after | 820,374 | 4.99 | 10,176 | 81 |
| 64 KiB | before | 1,248,146 | 52.51 | 145,046 | 85 |
| 64 KiB | after | 765,468 | 85.62 | 79,086 | 81 |
| 1 MiB | before | 1,719,977 | 609.65 | 2,112,109 | 86 |
| 1 MiB | after | 1,186,888 | 883.47 | 1,062,855 | 81 |

The optimized path reduced elapsed time by 7–39% in this run and reduced
allocated bytes by 30% for 4 KiB files, 45% for 64 KiB files, and 50% for 1 MiB
files. File-system timings vary with OS caching, so allocation results are the
more stable comparison.
