#!/bin/sh

sudo setcap 'cap_dac_read_search=ep' /usr/bin/observe-agent
