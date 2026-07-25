"""
Script drives the following workflow:
1. copies examples, generates tests for each subtask
2. records testgroup points, subtask, and public status
3. reports verdict, time, mem for each solution

`pygenlib` is available at https://github.com/KrisjanisP/pygenlib
"""
import logging
import os
import shutil
from tqdm import tqdm

from pygenlib.testgen import gen
from pygenlib.clean import clean
from pygenlib.tgyaml import record_tg, export_yaml
from pygenlib.report import report
from pygenlib import config as cfg

logging.basicConfig(
    level=logging.INFO,
    format="%(message)s",
    force=True,
)
logger = logging.getLogger(__name__)

# SCRIPT CONFIGURATION

cfg.set_task_id('cuska')

cfg.add_solution('../risin/cuska_kp_bfs_ok.cpp',is_model=True)
cfg.add_solution('../risin/cuska_ga.cpp')
cfg.add_solution('../risin/cuska_kp_bfs_ok.cpp')
cfg.add_solution('../risin/cuska_kp_never4ever.cpp')
cfg.add_solution('../risin/cuska_kp_greedy.cpp')

EXAMPLES_DIR = "./examples"

BOARDS_DIR = os.path.join(os.path.dirname(__file__), "boards")
EXTRA_FILES = {
    "N2.txt": os.path.join(BOARDS_DIR, "N2.txt"),
    "N3.txt": os.path.join(BOARDS_DIR, "N3.txt"),
    "N4.txt": os.path.join(BOARDS_DIR, "N4.txt"),
    "N5.txt": os.path.join(BOARDS_DIR, "N5.txt"),
    "solver.hpp": os.path.join(os.path.dirname(__file__), "solver.hpp"),
}

# ====================

def main():
    clean()
    gen_tests()
    gen_reports()



def gen_reports():
    logger.info("Generating reports")

    for sol in cfg.get_solution_paths():
        report(sol)

def gen_tests():
    logger.info("Generating tests")

    os.makedirs(cfg.get_tests_dir_path(),exist_ok=True)

    cp_examples()
    gen_subtask1()
    gen_subtask2()
    gen_subtask3()
    gen_subtask4()
    gen_subtask5()
    
    export_yaml()

tg = 0 # current testgroup count

def cp_examples():
    logger.info("Copying example files")

    tests_dir = cfg.get_tests_dir_path()
    for filename in os.listdir(EXAMPLES_DIR):
        example_path = os.path.join(EXAMPLES_DIR, filename)
        test_path = os.path.join(tests_dir, filename)
        shutil.copy(example_path, test_path)

def gen_subtask1():
    logger.info("Subtask 1 (N=2, 14 points)")
    global tg

    point_sum = 0
    for board_idx in tqdm(range(1,6+1), desc=f"N=2 boards 1 through 7"):
        tg += 1
        last = board_idx == 6
        points = 2 if not last else 14-point_sum
        record_tg(st=1, tg=tg, pts=points,public=last); point_sum += points
        gen(f"{tg:02}a", 2, board_idx, 1, 8, extra_files=EXTRA_FILES)
        gen(f"{tg:02}b", 2, board_idx, 1, 8, extra_files=EXTRA_FILES)
        gen(f"{tg:02}c", 2, board_idx/2+3, 1, 8, extra_files=EXTRA_FILES)
    
    assert point_sum == 14


def gen_subtask2():
    logger.info("Subtask 2 (N=3, 18 points)")
    global tg

    point_sum = 0
    for board_idx in tqdm(range(1,8+1), desc=f"N=3 boards 1 through 8"):
        tg += 1
        last = board_idx == 8
        points = 2 if not last else 18-point_sum
        record_tg(st=2, tg=tg, pts=points,public=last); point_sum += points
        gen(f"{tg:02}a", 3, board_idx, 2, 8, extra_files=EXTRA_FILES)
        gen(f"{tg:02}b", 3, board_idx, 2, 8, extra_files=EXTRA_FILES)
        gen(f"{tg:02}c", 3, board_idx, 2, 8, extra_files=EXTRA_FILES)

    assert point_sum == 18

def gen_subtask3():
    logger.info("Subtask 3 (N=3,4,5 with 1 cranberry, 20 points)")
    global tg

    point_sum = 0
    for (N,max_board_count) in tqdm([(3,8),(4,25),(5,20)], desc="N=3,4,5 with 1 craberry"):
        boards = min(max_board_count,5)
        for board_idx in range(1,boards+1):
            tg += 1
            last = board_idx == boards and N == 5
            points = 1 if not last else 20-point_sum
            assert points > 0
            record_tg(st=3, tg=tg, pts=points,public=last); point_sum += points
            gen(f"{tg:02}a", N, board_idx, 1, 1, extra_files=EXTRA_FILES)
            gen(f"{tg:02}b", N, board_idx, 1, 1, extra_files=EXTRA_FILES)
            if tg != 29:
                gen(f"{tg:02}c", N, board_idx, 1, 1, extra_files=EXTRA_FILES)
            else:
                gen(f"{tg:02}c", N, 1, 1, 1, extra_files=EXTRA_FILES)

    assert point_sum == 20

def gen_subtask4():
    logger.info("Subtask 4 (N=4, 22 points)")
    global tg
    
    point_sum = 0
    for board_idx in tqdm(range(1,11+1), desc=f"N=4 boards 1 through 22"):
        tg += 1
        last = board_idx == 11
        points = 2 if not last else 22-point_sum
        assert points > 0
        record_tg(st=4, tg=tg, pts=points,public=last); point_sum += points
        gen(f"{tg:02}a", 4, board_idx, 2, 8, extra_files=EXTRA_FILES)
        gen(f"{tg:02}b", 4, board_idx, 2, 8, extra_files=EXTRA_FILES)
        if tg != 38:
            gen(f"{tg:02}c", 4, board_idx, 2, 8, extra_files=EXTRA_FILES)
        else:
            gen(f"{tg:02}c", 4, 1, 2, 8, extra_files=EXTRA_FILES)

    assert point_sum == 22

def gen_subtask5():
    logger.info("Subtask 5 (Bez papildu ierobežojumiem, 26 points)")
    global tg

    point_sum = 0
    for board_idx in tqdm(range(1,13+1), desc=f"N=5 boards 1 through 13"):
        tg += 1
        last = board_idx == 13
        points = 2 if not last else 26-point_sum
        assert points > 0
        record_tg(st=5, tg=tg, pts=points,public=last); point_sum += points
        gen(f"{tg:02}a", 5, board_idx, 2, 8, extra_files=EXTRA_FILES)
        gen(f"{tg:02}b", 5, board_idx, 2, 8, extra_files=EXTRA_FILES)
        if tg != 45 and tg != 46:
            gen(f"{tg:02}c", 5, board_idx, 2, 8, extra_files=EXTRA_FILES)
        else:
            gen(f"{tg:02}c", 5, board_idx/2+6, 2, 8, extra_files=EXTRA_FILES)
    assert point_sum == 26

if __name__ == "__main__":
    main()
