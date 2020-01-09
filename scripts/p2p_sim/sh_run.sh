#!/bin/bash

for ((i=0; i<40; i++))
do
        if [ $i -lt  10 ] ; then
                echo node  ${i}
                nohup ./og -c configs/config_0${i}.toml -d data/d_0${i}  -l data/datadir_0${i} -n   run 2>&1 </dev/null &
        else
                echo node  ${i}
                nohup ./og -c configs/config_${i}.toml -d data/d_${i}  -l data/datadir_${i} -n   run  2>&1 </dev/null  &
        fi

done

