# Single point IPAM

This chain element is used to provide IPAM functionality for the nse-remote-vlan.

Per request allocates a single IP address in the given subnet and provides static routes. Request can set some exclude IP prefixes for the allocated IP.

The first IP address from the specified IP range and the broadcast IPv4 address will be witheld.
