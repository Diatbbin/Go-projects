import random
import string
import sys

def generate_instrument():
    """Generate a random instrument (e.g., stock symbol)"""
    return ''.join(random.choices(string.ascii_uppercase, k=1))

def generate_price():
    """Generate a random price between 1.00 and 100.00"""
    return round(random.uniform(1.00, 100.00), 2)

def generate_count():
    """Generate a random count between 1 and 100"""
    return random.randint(1, 100)

def generate_thread_id(num_threads):
    """Generate a random thread ID"""
    return random.randint(0, num_threads - 1)

def generate_unique_order_id(used_order_ids):
    """Generate a unique random order ID."""
    while True:
        order_id = random.randint(1, 100000000)  # Adjust range as needed
        if order_id not in used_order_ids:
            used_order_ids.add(order_id)
            return order_id

def generate_testcase(num_threads, num_testcases):
    testcases = []
    order_ids = {}  # map of order ID to thread ID
    used_order_ids = set()  # set of used order IDs (never deleted)
    thread_orders = {i: [] for i in range(num_threads)}  # map of thread ID to list of order IDs
    connected_threads = set()  # set of connected threads

    # First line: number of threads
    print(str(num_threads))

    # Connect all threads
    for thread_id in range(num_threads):
        print(f"{thread_id} o")  # open
        connected_threads.add(thread_id)

    # Generate test cases
    for _ in range(num_testcases):
        thread_id = random.choice(list(connected_threads))
        action = random.choice(['B', 'S', 'C'])

        if action == 'B' or action == 'S':  # buy/sell
            order_id = generate_unique_order_id(used_order_ids)
            instrument = generate_instrument()
            price = generate_price()
            count = generate_count()
            print(f"{thread_id} {action} {order_id} {instrument} {price} {count}")
            order_ids[order_id] = thread_id
            thread_orders[thread_id].append(order_id)
        elif action == 'C':  # cancel
            if thread_orders[thread_id]:
                order_id = random.choice(thread_orders[thread_id])
                print(f"{thread_id} C {order_id}")
                thread_orders[thread_id].remove(order_id)
                del order_ids[order_id]

    # Disconnect all threads
    for thread_id in connected_threads:
        print(f"{thread_id} x")  # disconnect

    return '\n'.join(testcases)

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 test_case.py <num_threads> <num_testcases>")
        sys.exit(1)

    try:
        num_threads = int(sys.argv[1])
        num_testcases = int(sys.argv[2])
        if num_threads <= 0 or num_testcases <= 0:
            print("Error: Number of threads and test cases must be positive integers.")
            sys.exit(1)
    except ValueError:
        print("Error: Number of threads and test cases must be integers.")
        sys.exit(1)

    generate_testcase(num_threads, num_testcases)
