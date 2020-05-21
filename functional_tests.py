# Project: raft-lite
# ---------------------
# Authors:
#   Minjian Chen 813534
#   Shijie Liu   813277
#   Weizhi Xu    752454
#   Wenqing Xue  813044
#   Zijun Chen   813190

import sys
import argparse
from typing import *
import subprocess
import time
from multiprocessing import Pool


class bcolors:
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'


def main():
    # parse arguments
    parser = argparse.ArgumentParser(
        description='Helper tool for running functional tests')
    parser.add_argument('--times', type=int, default=5,
                        help='number of times each test case will be run')
    parser.add_argument('--parallel', type=int, default=2,
                        help='number of tests allowed to run in parallel')
    parser.add_argument('--timeout', type=int, default=5*60,
                        help='time limit for each tests (in second)')
    parser.add_argument('--single_testcase', type=int, default=0,
                        help='single testcase number')
    args = parser.parse_args()
    print(args)
    if not run_tests(args.times, args.parallel, args.timeout, args.single_testcase):
        sys.exit(1)

# build the go binary executable


def build_executable() -> None:
    res = subprocess.run(["go", "build", "."],
                         timeout=20, stdout=subprocess.PIPE,
                         stderr=subprocess.STDOUT, encoding="utf-8")
    if res.returncode != 0:
        raise RuntimeError("Build Fail"+res.stdout)

# Return values are (passed, timeout, output)

# run raft-lite programme supplying with given running arguments


def execute_raft_lite(args: List[str], timeout: int) -> Tuple[bool, bool, str]:
    try:
        res = subprocess.run(['./raft-lite']+args,
                             timeout=timeout, stdout=subprocess.PIPE,
                             stderr=subprocess.STDOUT, encoding="utf-8")
    except subprocess.TimeoutExpired as e:
        return (False, True, str(e.stdout))
    success = res.returncode == 0
    return (success, False, str(res.stdout))


# run a single test case and returns the results
def run_single_test(task: Tuple[int, int, int]) -> Tuple[bool, bool, str]:
    time.sleep(0.1)
    (passed, timeout, tests_list) = execute_raft_lite(
        ["functionaltest", "run", "-n", str(task[0])], task[2])
    time.sleep(0.1)
    if passed:
        print(".", end="", flush=True)
    elif timeout:
        print(bcolors.FAIL+"*"+bcolors.ENDC, end="", flush=True)
    else:
        print(bcolors.FAIL+"X"+bcolors.ENDC, end="", flush=True)

    return (passed, timeout, tests_list)


# run a batch of tests that runs mutiple times and runs in parallel, or it might runs single testcases
def run_tests(times: int, parallel: int, timeout: int, single_testcase: int) -> bool:
    build_executable()
    (ok, _, out) = execute_raft_lite(["functionaltest", "count"], timeout)
    if not ok:
        raise RuntimeError("Unable to load test cases")
    (ok, _, tests_list) = execute_raft_lite(
        ["functionaltest", "list"], timeout)
    if not ok:
        raise RuntimeError("Unable to load test cases")
    tests_n = int(out)
    if single_testcase != 0:
        tasks = [(single_testcase+1, j+1, timeout) for j in range(times)]
    else:
        tasks = [(i+1, j+1, timeout)
                 for i in range(tests_n) for j in range(times)]
    print("="*40)
    print("Start running tests")
    print(tests_list)
    print("="*40)

    # run in parallel
    print("")
    with Pool(parallel) as pool:
        results = pool.map(run_single_test, tasks)
    print("")
    print("="*40)

    # print results
    stats = [[0, 0, 0] for _ in range(tests_n)]
    for ((test_id, _, _), (ok, timeout, _)) in zip(tasks, results):
        if single_testcase != 0:
            test_id = single_testcase
        if ok:
            stats[test_id-1][0] += 1
        elif timeout:
            stats[test_id-1][1] += 1
        else:
            stats[test_id-1][2] += 1

    for i in range(len(stats)):
        print("{:4}: ".format(i+1), end="")
        if stats[i][0] == times:
            print(bcolors.OKGREEN+"OK"+bcolors.ENDC)
        else:
            print(bcolors.FAIL+"FAIL"+bcolors.ENDC, end=" ")
            print("ok: {}, timeout: {}, fail: {}".format(
                stats[i][0], stats[i][1], stats[i][2]))

    print("")
    print("="*40)
    all_good = True

    # print output of failed cases
    for (ok, timeout, output) in results:
        if not ok:
            print("~"*40)
            print(output)
            print("")
            print("~"*40)
            all_good = False

    return all_good


if __name__ == "__main__":
    main()
