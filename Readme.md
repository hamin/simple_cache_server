# Simple Cache Server

This is a simple key value store (cache server) written in go as an __exercise__. The Spec is based on a [TopCoder challenge](http://community.topcoder.com/tc?module=ProjectDetail&pj=30046225).

To test and use this server, just use Telnet as described in the spec. The implementation is functinonal and complete based on the spec. It is definitely not perfect. __Please don't use this in production!__

Take a look and have fun and implement one yourself if you're inspired :)

##Spec

The cache server communicates with clients via TCP
The cache server provides five commands set, get, delete, stats and quit
The cache server handles multiple connections
The cache server stores data on memory and does not persist it
See “Examples” section to know how the commands work. Please note that the cache server should maintain the connection until ‘quit’ command is sent.

##Commands

__set command__

* Store data. Overwrite data if the data exists.

_client request_

set \<key>\r\n

\<data>\r\n

- <key> is the key to store the data. The size must be less than 250 characters.

- \<data> is data to store. The size must be less than 8KB.

The character range of key and data is the following ASCII characters.

* a-z
* A-Z
* 0-9
* ! # $ % & ' " * + - / = ? ^ _ ` { | } ~
* ( ) < > [ ] : ; @ , .
* space (just for data)

_server response_

The server sends the string "STORED\r\n" to indicate success.

__get command__

Retrieve data. Take one or more keys and returns all found items.

_client request_
get <key>*\r\n

- <key>* means one or more key strings separated by whitespace.

_server response_
VALUE <key>\r\n
\<data>\r\n

- <key> is the key for the item being sent.
- \<data> is the data for this item.
After the server sends all the items, the server sends the string "END\r\n"

__delete command__

Remove an item from the cache, if it exists.

_client request_
delete <key>\r\n

- <key> is the key of the item the client would like the server to delete

_server response_

- "DELETED\r\n" to indicate success.
- "NOT_FOUND\r\n" to indicate that the item was not found.

__stats command__
Output statistics and settings below.

+ cmd_get
	+ Cumulative number of get
	+ “get” command with multiple keys is counted by number of keys
	+ e.g.
		+ “get appirio topcoder” command is counted as 2 get commands.

+ cmd_set
	+ Cumulative number of set

+ get_hits
	+ Number of get command for items stored

+ get_misses
	+ Number of get command for items not stored

+ delete_hits
	+ Number of delete command for items stored

+ delete_misses
	+ Number of delete command for items not stored

+ curr_items
	+ Current number of items stored

+ limit_items
	+ Number of items this server is allowed to store.
	+ This number does not change while the server process is running.

_client request_

stats\r\n

_server response_

\<stat> <number>\r\n
\<stat> <number>\r\n

...

(See Examples section)
The server terminates this list with the line
“END\r\n”

__quit command__

Terminate connection to the server

_client request_
quit\r\n

_server response_
No response. The connection is just closed.

##Errors

In case of errors, the server sends an error string.
- "ERROR <error>\r\n"
<error> should be a human-readable and easy to detect the reason.
Please note that the server just ignores an empty line that has only "\r\n".
Extensibility

We would like to support new commands with less effort in the future, so extensibility to add new commands is important.
Examples

Blue characters are responses from the server.

```
telnet localhost 11212
Trying 127.0.0.1...
Connected to localhost.
Escape character is '^]'.
set sushi
delicious
STORED
set topcoder
fun
STORED
get sushi topcoder
VALUE sushi
delicious
VALUE topcoder
fun
END
get sushi
VALUE sushi
delicious
END
delete sushi
DELETED
get sushi
END
get topcoder
VALUE topcoder
fun
END
get topcoder sushi
VALUE topcoder
fun
END
delete sushi
NOT_FOUND
stats
cmd_get 7
cmd_set 2
get_hits 5
get_misses 2
delete_hits 1
delete_misses 1
curr_items 1
limit_items 65535
END
quit
Connection closed by foreign host.
```

##Command-line options

The server provides the following command-line options

__Port__

* The port the server listens on
* “-port” option
* The default is 11212

__Items__

* Total number of items the server can store
“-items” option
* The default is 65535

__Example__

* go run <your cache server>.go -port 11213 -items 1024
Signal Handling

Please handle the following signal and terminate the server after all commands being requested are processed.
os.Interrupt (syscall.SIGINT)