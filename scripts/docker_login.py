#!/usr/bin/env python

import sys
import os
import pathlib

sys.path.append(os.path.dirname(__file__))
## resolutions for ModuleNotFoundError s in the current directory !?
# sys.path.append(str(pathlib.Path(__file__).resolve().parent))

import logging
import json

import argparse
import string

from urllib.parse import urlparse
from typing import Any, Callable
import copy

# logging.basicConfig(encoding='utf-8', level=logging.INFO)
# logger = logging.getLogger(__name__)

logging.basicConfig(format='%(asctime)s,%(msecs)03d %(levelname)-8s [%(filename)s:%(lineno)d] %(message)s',
   datefmt='%Y-%m-%d:%H:%M:%S',
   level=logging.INFO)


logger = logging.getLogger(__name__)


parser = argparse.ArgumentParser(
 description=(
   "Parses and manages a docker configuration file as input (as argument or stdin)."
   )
)


user_home = str(pathlib.Path.home())

parser.add_argument('--version', action='version', version='%(prog)s 0.1',
                   help="print version info")
parser.add_argument('--infile', '-i', nargs='?',
                   # type=argparse.FileType('r'),
                   # default=sys.stdin,
                   default="~/.docker/config.json",
                   help="input file containing docker configuration as json. Defaults to ~/.docker/config.json")
parser.add_argument('--outfile', '-o', nargs='?', type=argparse.FileType('w'), default=sys.stdout,
                   help="output file (or stdout) containing docker configuration as json")
parser.add_argument('--registry', '-r', required=True,
                   help="address of the registry. If an ECR registry is passed, uses aws cli to log in..")
parser.add_argument("--modify-inplace", '-m', action="store_true", default=False,
                   help="write the modified registry back into the \"infile\".")
parser.add_argument("--verbose", '-v', action="store_true", default=False,
                   help="print successful logins and config changes to stderr")
# parser.add_argument('--region', '-R', required=True,
#                    help="region of the ECR registry ..")

args = parser.parse_args()

config: Any = None


def prog_exit(msg: string):
   logging.error(f'{msg}. Exiting!')
   exit(1)


input_path = pathlib.Path(args.infile).expanduser().resolve()
if not input_path.exists():
   input_path.parent.mkdir(parents=True, exist_ok=True)
   with input_path.open("w", encoding ="utf-8") as f:
       f.write("{}")
   if args.verbose:
       print(f"Created new docker config file: {input_path}", file=sys.stderr)


with open(str(input_path), 'r') as f:
   config = json.load(f)


config = config | {}


def get_auth_for_registry(conf: dict, registry: string):
   ret = conf

   if 'auths' not in ret:
       ret['auths'] = {}
   ret = ret['auths']

   if registry not in ret:
       ret[registry] = {'auth': {}}

   ret = ret[registry]
   return ret


def get_aws_auth_token(registry: string):
   import subprocess
   aws_env = os.environ.copy()

   region = registry.split(".")[3]
   result = subprocess.run(["aws", "ecr", "get-login-password", "--region", region], capture_output=True, check=False, env=aws_env)
   if result.stderr:
       logger.error(f'Error in get_aws_auth_token: {result.stderr.decode("utf-8")}')
       return None
   return result.stdout.decode(encoding="UTF-8")


auth = get_auth_for_registry(config, args.registry)
aws_token = get_aws_auth_token(args.registry)
# logger.warning(f'aws_token={aws_token}')
if not aws_token:
   prog_exit("Could not get aws ecr login token.")

if args.verbose:
   print(f"Successfully obtained AWS ECR login token for registry: {args.registry}", file=sys.stderr)


import base64
b64encoding = 'ascii'
## ascii encoding would be proper??
auth['auth'] = base64.b64encode(f'AWS:{aws_token}'.encode(b64encoding)).decode(b64encoding)

if args.verbose:
   print(f"Updated docker config with authentication for registry: {args.registry}", file=sys.stderr)


if args.modify_inplace:
   with open(str(input_path), 'w') as f:
       json.dump(config, f, indent=4)
   logger.debug(f'Competed processing config to outfile={input_path}')
   if args.verbose:
       print(f"Config written to: {input_path}", file=sys.stderr)
else:
   with args.outfile as f:
       json.dump(config, f, indent=4)
   logger.debug(f'Competed processing config to outfile={args.outfile}')
   if args.verbose:
       print(f"Config written to: {args.outfile.name if hasattr(args.outfile, 'name') else 'stdout'}", file=sys.stderr)

if args.verbose:
   print(f"Successfully logged into registry: {args.registry}", file=sys.stderr)

