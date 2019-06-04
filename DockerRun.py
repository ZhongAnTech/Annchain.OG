import os
import argparse
import time
import subprocess

def deploy():
    parser = argparse.ArgumentParser()
    parser.add_argument("-m", help="node mode, 'n' for normal node, 'p' for private node, 's' for solo node")
    parser.add_argument("-b", help="bootnode if this tag exists", action="store_true")
    parser.add_argument("-c", help="join consensus with a consensus filename")

    args = parser.parse_args()
    print(args)

    time_format = "%Y-%m-%d_%H-%M-%S"
    cur_time = time.strftime(time_format, time.localtime())
    os.mkdir("/deployment/Node " + cur_time)

    if args.m != 'n' and args.m != 'p':
        print("invalid node mode, must be n or p")
        return 

    if args.m == 'n' or args.m == None:
        subprocess.call(["./DockerRun.sh"])
        return

    if args.m == 's':
        # solo node 
        # close annsensus and let auto_client produce sequencers automatically
        return

    if args.m == 'p':

        pass

    # check if it is bootnode
    if args.b:
        # generate onode
        # reset onode
        pass

     


if __name__ == "__main__":
    deploy()



