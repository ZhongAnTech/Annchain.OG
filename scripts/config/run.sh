killall -v og
rm -rvf logs

for ((i=0; i<4; i++))
do
        if [ $i -lt  10 ] ; then
                echo node  ${i}
                nohup OG/build/og -c private/node_${i}/config.toml   -l logs/log${i} -n   run 2>&1 </dev/null &
        else
                echo node  ${i}
                nohup ./og -c configs/config_${i}.toml -d data/d_${i}  -l data/datadir_${i} -n   run  2>&1 </dev/null  &
        fi

done
