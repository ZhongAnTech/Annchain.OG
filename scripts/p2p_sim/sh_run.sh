#!/bin/bash

for ((i=0; i<=100; i++)) 
do
	if [ $i -lt  10 ] ; then
		echo node  ${i}
		../build/og -c configs/config_0${i}.toml -d data/d_0${i}  -l data/datadir_0${i} -n -M  run 1>/dev/null  & 
	else
		echo node  ${i}  
		../build/og -c configs/config_${i}.toml -d data/d_${i}  -l data/datadir_${i} -n -M  run  1>/dev/null  &
	fi	

done
