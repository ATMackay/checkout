# Checkout 

## System Design 

The Checkout server is stateless and exposes a RESTful http interface.

## Components

* Go HTTP server exposing a RESTful API built with [httprouter](https://github.com/julienschmidt/httprouter).
* [Prometheus](https://prometheus.io/) metrics server endpoint.