## Tips for reading and manipulating logs.

### How to understand logs

1. Every log line starts with timestamp, which is helpful when tracing request in the application chain
2. Most of the log lines contains `[type:NetworkService]` or `[type:NetworkServiceRegistry]` which helps to separate `request` logic from `registration` logic
3. Most of the log lines contains `[id:'some value']` which is helpful when tracing request in application chain
4. It is useful to transform logs using `grep` utility to discard 'noise' logs that is not needed for analyzing(examples will be in section `Useful scripts`)
5. It is helpful to install plugin on IDE which is colorizing logs. (for example, ANSI highlighter for Goland)
6. Message `an error during getting metadata from context: metadata is missed in ctx` is a good criteria for separating ordinary request from refresh request(refresh doesn't have metadata)

### Useful scripts

1. Applied to folder containing resulting logs. It removes lines related to Jaeger and remove lines with just spans. Also, it changes file extension to one suitable for highlighters
```bash
for filename in *.logs; do
    cat "${filename}" | grep -v "Reporting span" | grep -v "Jaeger" > "$(echo "${filename}" | sed "s/\.logs/\.log/g")"
    rm "${filename}"
done
```

2. It is useful to grep logs for extracting specific information that you want - by id, type, loglevel etc. For example: 
- get only networkService type lines
```bash
grep -w type:networkService some_log_file.log > another_log_file.log
```
- get logs **without** any NetworkServiceEndpointRegistry type lines
```bash
grep -w -v type:NetworkServiceEndpointRegistry some_log_file.log > another_log_file.log
```
- get all lines with specific request id (random id in example)
```bash
grep -w id:d7bb2d77-7cd4-44ad-902b-5e392852d93a-final-endpoint some_log_file.log > another_log_file.log
```