# BuildNumberService

Configuration
=============
The build number service's configuration file is a brief YAML document that holds the following settings.

* ``pidfile`` Path to the pidfile the build number service will create.
* ``dbpath`` Path to the sqlite3 database file created by the service.
* ``port`` Port the REST api server listens on.
* ``variablename`` The variable name used for the build number for all jobs

Usage
=====
By default the build number searvice tries to read its configuration file at /etc/bns.yaml.  You can override this on the bns command line with the "-config" flag.

Getting Build Numbers
=====================

For project banana, with the following configuration file:
````
pidfile: /var/run/bns.pid
dbpath: /var/lib/bns/build_numbers.sqlite3
port: 80
variablename: MY_BUILD_NUMBER
````

To get a new build number: ``http://numbers.banana.com/banana/inc``  (returns ``MY_BUILD_NUMBER=1``)

To get the current build number: ``http://numbers.banana.com/banana`` (returns ``MY_BUILD_NUMBER=1``)

Append ``/bash``, ``/json``, or ``/yaml`` to the above URLs to get the results ``MY_BUILD_NUMBER=1``, ``{"MY_BUILD_NYMBER": 1}``, and ``MY_BUILD_NUMBER: 1`` respectively.

The format specifiers may also be used with the ``/inc`` path in the first example.

To set the current build number to 54, POST the following url: ``http://numbers.banana.com/banana/54``.  Note that this url currently only returns a Bash formatted response.
