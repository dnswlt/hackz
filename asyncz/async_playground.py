

import asyncio
import time

STARTED = None

def log(msg):
    reltime = (time.perf_counter_ns() - STARTED) // 1000000
    print(f"{reltime: 8}: {msg}")

async def foo(delay: int):
    log("foo: started")
    await asyncio.sleep(delay)
    log("foo: done")


async def taskgrp():
    async with asyncio.TaskGroup() as tg:
        task1 = tg.create_task(foo(1))
        task2 = tg.create_task(foo(2))
    log("taskgrp: done")

def main():
    global STARTED
    STARTED = time.perf_counter_ns()
    # asyncio.run(foo(1))
    asyncio.run(taskgrp())
    
if __name__ == "__main__":
    main()