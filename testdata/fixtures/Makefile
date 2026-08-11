# Fixture Makefile for bounded make expansion tests.
GREETING = building

build:
	echo $(GREETING)

a: b
	echo a-body

b: a
	echo b-body

parent1: shared
parent2: shared

shared:
	echo shared

root: parent1 parent2
