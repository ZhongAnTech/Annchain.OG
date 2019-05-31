import os
import argparse
import time

parser = argparse.ArgumentParser()
parser.add_argument("-m", help="node mode, 'n' for normal node, 'p' for private node")
parser.add_argument("-b", help="bootnode if this tag exists", action="store_true")
parser.add_argument("-c", help="join consensus with a consensus filename")

args = parser.parse_args()
print(args)

time_format = "%Y-%m-%d_%H-%M-%S"
cur_time = time.strftime(time_format, time.localtime())
os.mkdir("Node " + cur_time)



