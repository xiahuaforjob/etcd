# etcd-dump-logs

`etcd-dump-logs` dumps the log from data directory.

## Installation

Install the tool by running the following command from the etcd source directory.

```
  $ go install -v ./tools/etcd-dump-logs
```

The installation will place executables in the $GOPATH/bin. If $GOPATH environment variable is not set, the tool will be installed into the $HOME/go/bin. You can also find out the installed location by running the following command from the etcd source directory. Make sure that $PATH is set accordingly in your environment.

```
  $ go list -f "{{.Target}}" ./tools/etcd-dump-logs
```

Alternatively, instead of installing the tool, you can use it by simply running the following command from the etcd source directory.

```
  $ go run ./tools/etcd-dump-logs
```

## Usage

The following command should output the usage per the latest development.

```
  $ etcd-dump-logs --help
```

An example of usage detail is provided below.

```
Usage:

  etcd-dump-logs [data dir]
    * Data dir is where the snapshots and WAL logs are located. The structure of the data dir should look like this:
        - data_dir/member
            - data_dir/member/snap
            - data_dir/member/wal
                - data_dir/member/wal/0000000000000000-0000000000000000.wal

Flags:
  -wal-dir string
      If set, dumps WAL from the informed path, rather than following the
      standard 'data_dir/member/wal/' location
  -entry-type string
    	If set, filters output by entry type. Must be one or more than one of:
	    ConfigChange, Normal, Request, InternalRaftRequest,
	    IRRRange, IRRPut, IRRDeleteRange, IRRTxn,
	    IRRCompaction, IRRLeaseGrant, IRRLeaseRevoke
  -start-index uint
    	The index to start dumping (inclusive)
      If unspecified, dumps from the index of the last snapshot.
  -end-index uint
      The index to stop dumping (exclusive)
  -start-snap string
    	The base name of snapshot file to start dumping
  -stream-decoder string
    	The name and arguments of an executable decoding tool, the executable
    	must process hex encoded lines of binary input (from etcd-dump-logs)
	    and output a hex encoded line of binary for each input line
```
#### etcd-dump-logs -entry-type <ENTRY_TYPE_NAME(S)> [data dir]

Filter entries by type from WAL log.

```
$ etcd-dump-logs -entry-type IRRTxn /tmp/datadir
Snapshot:
empty
Start dupmping log entries from snapshot.
WAL metadata:
nodeID=0 clusterID=0 term=0 commitIndex=0 vote=0
WAL entries:
lastIndex=34
term	     index	type	data
   7	        13	norm	ID:8 txn:<success:<request_delete_range:<key:"a" range_end:"k8s\000\n\025\n\002v1\022\017RangeAllocation\022#\n\022\n\000\022\000\032\000\"\000*\0002\0008\000B\000z\000\022\01310.0.0.0/16\032\000\032\000\"\000" > > failure:<request_delete_range:<key:"a" range_end:"k8s\000\n\025\n\002v1\022\017RangeAllocation\022#\n\022\n\000\022\000\032\000\"\000*\0002\0008\000B\000z\000\022\01310.0.0.0/16\032\000\032\000\"\000" > > >

Entry types (IRRTxn) count is : 1

$ etcd-dump-logs -entry-type ConfigChange,IRRCompaction /tmp/datadir
Snapshot:
empty
Start dupmping log entries from snapshot.
WAL metadata:
nodeID=0 clusterID=0 term=0 commitIndex=0 vote=0
WAL entries:
lastIndex=34
term	     index	type	data
   1	         1	conf	method=ConfChangeAddNode id=2
   2	         2	conf	method=ConfChangeRemoveNode id=2
   2	         3	conf	method=ConfChangeUpdateNode id=2
   2	         4	conf	method=ConfChangeAddLearnerNode id=3
   8	        14	norm	ID:9 compaction:<physical:true >

Entry types (ConfigChange,IRRCompaction) count is : 5
```
#### etcd-dump-logs -stream-decoder <EXECUTABLE_DECODER> [data dir]

Decode each entry based on logic in the passed decoder. Decoder status and decoded data are listed in separated tab/columns in the output. For parsing purpose, the output from decoder are expected to be in format of "<DECODER_STATUS>|<DECODED_DATA>". Please refer to [decoder_correctoutputformat.sh] as an example.

However, if the decoder output format is not as expected, "decoder_status" will be "decoder output format is not right, print output anyway", and all output from decoder will be considered as "decoded_data"


```
$ etcd-dump-logs -stream-decoder decoder_correctoutputformat.sh  /tmp/datadir
Snapshot:
empty
Start dupmping log entries from snapshot.
WAL metadata:
nodeID=0 clusterID=0 term=0 commitIndex=0 vote=0
WAL entries:
lastIndex=34
term	     index	type	data	decoder_status	decoded_data
   1	         1	conf	method=ConfChangeAddNode id=2	ERROR	jhjaajjjahjbbbjj
   3	         2	norm	noop	OK	jhjjabjjaajfbfgjfagdfhcjbbahgbbbfhfegibbcabbfhffbbbcbbfhfibbcaebbbgiffbbedgdbhjacbjjchjjdjjjdhjiejjjehjafjjjfhjjgjjjghjahjjajjhhjajj
   3	         3	norm	method=QGET path="/path1"	OK	jhjaabjdeadgdeedaajfbfgjfagdfhcabbacgbbbcjbbcabbcabbbcbbcbbbcaebbbccbbedgdbhjjcbjjchjjdjjjdhjiejjjehjafjjjfhjjgjjjghjahjjajjhhjajj
   7	         4	norm	ID:8 txn:<success:<request_delete_range:<key:"a" range_end:"b" > > failure:<request_delete_range:<key:"a" range_end:"b" > > > 	OK	jhjhcbadabjhaajfjajafaabjafbaajhaajfjajafaabjafb
   8	         5	norm	ID:9 compaction:<physical:true > 	ERROR	jhjicajbajja
   9	         6	norm	ID:10 lease_grant:<TTL:1 ID:1 > 	ERROR	jhjadbjdjhjaajja
  12	         7	norm	ID:13 auth_enable:<> 	ERROR	jhjdcbcejj
  27	         8	norm	???	ERROR	cf
Entry types () count is : 8
```

```
$ etcd-dump-logs -stream-decoder decoder_wrongoutputformat.sh  /tmp/datadir
Snapshot:
empty
Start dupmping log entries from snapshot.
WAL metadata:
nodeID=0 clusterID=0 term=0 commitIndex=0 vote=0
WAL entries:
lastIndex=34
term	     index	type	data	decoder_status	decoded_data
   1	         1	conf	method=ConfChangeAddNode id=2	decoder output format is not right, print output anyway	jhjaajjjahjbbbjj
   3	         2	norm	noop	decoder output format is not right, print output anyway	jhjjabjjaajfbfgjfagdfhcjbbahgbbbfhfegibbcabbfhffbbbcbbfhfibbcaebbbgiffbbedgdbhjacbjjchjjdjjjdhjiejjjehjafjjjfhjjgjjjghjahjjajjhhjajj
   3	         3	norm	method=QGET path="/path1"	decoder output format is not right, print output anyway	jhjaabjdeadgdeedaajfbfgjfagdfhcabbacgbbbcjbbcabbcabbbcbbcbbbcaebbbccbbedgdbhjjcbjjchjjdjjjdhjiejjjehjafjjjfhjjgjjjghjahjjajjhhjajj
   7	         4	norm	ID:8 txn:<success:<request_delete_range:<key:"a" range_end:"b" > > failure:<request_delete_range:<key:"a" range_end:"b" > > > 	decoder output format is not right, print output anyway	jhjhcbadabjhaajfjajafaabjafbaajhaajfjajafaabjafb
   8	         5	norm	ID:9 compaction:<physical:true > 	decoder output format is not right, print output anyway	jhjicajbajja
   9	         6	norm	ID:10 lease_grant:<TTL:1 ID:1 > 	decoder output format is not right, print output anyway	jhjadbjdjhjaajja
  12	         7	norm	ID:13 auth_enable:<> 	decoder output format is not right, print output anyway	jhjdcbcejj
  27	         8	norm	???	decoder output format is not right, print output anyway	cf
Entry types () count is : 8

```
####  etcd-dump-logs -start-index <INDEX NUMBER> [data dir]

Only shows WAL log entries after the specified start-index number, inclusively.

```
$ etcd-dump-logs -start-index 31  /tmp/datadir
Start dumping log entries from index 31.
WAL metadata:
nodeID=0 clusterID=0 term=0 commitIndex=0 vote=0
WAL entries:
lastIndex=34
term	     index	type	data
  25	        31	norm	ID:26 auth_role_get:<role:"role3" >
  26	        32	norm	ID:27 auth_role_grant_permission:<name:"role3" perm:<permType:WRITE key:"Keys" range_end:"RangeEnd" > >
  27	        33	norm	ID:28 auth_role_revoke_permission:<role:"role3" key:"key" range_end:"rangeend" >
  27	        34	norm	???
Entry types () count is : 4
```

####  etcd-dump-logs -start-index <INDEX NUMBER> -end-index <INDEX NUMBER> [data dir]

Only shows WAL log entries from the specified start-index number (inclusively) to the specified end-index number (exclusively).

```
$ etcd-dump-logs -start-index 930 -end-index 932  /tmp/datadir
Start dumping log entries from index 930.
WAL metadata:
nodeID=0 clusterID=0 term=5 commitIndex=2448 vote=0
WAL entries: 2
lastIndex=931
term	     index	type	data
   3	       930	norm	header:<ID:11010058442592651283 > put:<key:"key7" value:"923" >
   3	       931	norm	header:<ID:6577953459306661672 > put:<key:"key8" value:"924" >

Entry types (Normal,ConfigChange) count is : 2
```

[decoder_correctoutputformat.sh]: ./testdecoder/decoder_correctoutputformat.sh
