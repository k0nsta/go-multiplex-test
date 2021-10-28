# go-multiplex-test

Simple multiplexer handles one endpoint `/collector`.

The service accepts URLs returns content of given URLs or errors.

Limits (configurable):

    - 100 simultaneous incoming connects;
    - 4 simultaneous outcoming connects for each incoming request;
    - 20 urls per incoming request.

