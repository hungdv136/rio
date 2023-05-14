#!/bin/bash

until mysqladmin ping -s -h db -P 3306 -uadmin -ppassword; do
  sleep 1
done

make test
