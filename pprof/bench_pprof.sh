
# 运行解码基准测试
go test -bench=Decode -benchmem -cpuprofile=cpu_decode.prof -memprofile=mem_decode.prof -trace=trace_decode.out

# 运行编码基准测试
go test -bench=Encode -benchmem -cpuprofile=cpu_encode.prof -memprofile=mem_encode.prof -trace=trace_encode.out

# 查看基准测试结果
go tool pprof -http=:8080 cpu_decode.prof
go tool pprof -http=:8080 mem_decode.prof
go tool trace trace_decode.out

go tool pprof -http=:8080 cpu_encode.prof
go tool pprof -http=:8080 mem_encode.prof
go tool trace trace_encode.out

# 转换为svg
go tool pprof -svg cpu_decode.prof > cpu_decode.svg
go tool pprof -svg mem_decode.prof > mem_decode.svg
go tool pprof -svg cpu_encode.prof > cpu_encode.svg
go tool pprof -svg mem_encode.prof > mem_encode.svg


# 查看trace
go tool trace trace_decode.out
go tool trace trace_encode.out

# 查看 text
go tool pprof -text cpu_decode.prof
go tool pprof -text mem_decode.prof
go tool pprof -text cpu_encode.prof
go tool pprof -text mem_encode.prof
