
Watchserver
===========

A simple tcp server that notifies all clients whenever a file is modified.
Includes a simple client which blinks the lights on a keyboard when a it
recieves any notification from the server.

## Installation

	go get github.com/lelandbatey/watchserver/...
	# Allow watchclient to run as root so it can blink the lights
	sudo chown root $(which watchclient) && sudo chmod u+s $(which watchclient)
	watchserver /tmp/
	watchclient
	echo "what" >> /tmp/example

