# ippool.IPPool

It is a red–black tree containing IPv4/IPv6 address networks.
IP address represents as two uint64 numbers (high and low 64 bits of 128 bit IPv6 address). Each node of RB tree is bounds of IP range. 


## Performance

Performance results for IPPool, RoaringBitmap and PrefixPool. Each iteration pulls P2P address pair for pool with 1000 excluded subnets.

 BenchmarkIPPool | ops | ns/op | B/op | allocs/op 
 ----------- | ----------- | ----------- | ----------- | ----------- 
BenchmarkIPPool/IPPool | 566 | 2161121 | 440206 | 22007
BenchmarkIPPool/RoaringBitmap | 1359 | 824205 | 218841 | 11106
BenchmarkIPPool/PrefixPool | 2 | 1803298500 | 1001995744 | 18689007
  |   |   |   |  
BenchmarkIPPool/IPPool-2 | 1009 | 1079308 | 440223 | 22008
BenchmarkIPPool/RoaringBitmap-2 | 2985 | 351601 | 218850 | 11106
BenchmarkIPPool/PrefixPool-2 | 2 | 1355259309 | 1021856016 | 19835510
  |   |   |   |  
BenchmarkIPPool/IPPool-4 | 1126 | 1005249 | 440240 | 22008
BenchmarkIPPool/RoaringBitmap-4 | 5494 | 193221 | 218847 | 11106
BenchmarkIPPool/PrefixPool-4 | 2 | 1365976629 | 1021857340 | 19835525
  |   |   |   |  
BenchmarkIPPool/IPPool-8 | 1011 | 1095871 | 440265 | 22009
BenchmarkIPPool/RoaringBitmap-8 | 6398 | 163107 | 218842 | 11106
BenchmarkIPPool/PrefixPool-8 | 2 | 1319444212 | 1012589580 | 19802446
  |   |   |   |  
BenchmarkIPPool/IPPool-16 | 996 | 1102098 | 440250 | 22008
BenchmarkIPPool/RoaringBitmap-16 | 7626 | 179327 | 218851 | 11106
BenchmarkIPPool/PrefixPool-16 | 2 | 1408024424 | 1012589068 | 19802437
