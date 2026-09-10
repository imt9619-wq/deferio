# Deferio
### Deferio is a two module anti cheat, with diohandler and dioAntiCheat
##### (Still in progress)

## DioHandler
### Handle packets
- intercept incoming and going packets by custom server listener - forward a copied packets to anti cheat server for packet analysis ![]()
- share same tx with session's packet handler by wrapping the session handler with our own handler - mainly for player movement
- normal player handler - hit cancellation for probable reach or killaura

## DioAntiCheat

## How a server network work with deferio
![deferio.png](/docs/deferio.png)