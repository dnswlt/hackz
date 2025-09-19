from collections import defaultdict
import csv
from io import StringIO
from dataclasses import dataclass
import sys

# Using a dataclass for better structure and type safety.
@dataclass
class Process:
    """Represents a type of process to be deployed."""
    name: str
    memory: float
    cpu: float
    instances: int

# Also defining a Pod dataclass for consistency.
@dataclass
class Pod:
    """Represents a single instance of a Process."""
    name: str
    instance_name: str
    memory: float
    cpu: float


class Node:
    """Represents a single EC2 instance with allocatable resources."""
    def __init__(self, id, cpu_capacity, mem_capacity):
        self.id = id
        self.cpu_capacity = cpu_capacity
        self.mem_capacity = mem_capacity
        self.cpu_remaining = cpu_capacity
        self.mem_remaining = mem_capacity
        self.pods = []
        self.hosted_processes = set()

    def can_fit(self, pod: Pod):
        """Checks if a pod can fit based on resources and anti-affinity."""
        # 1. Check resource availability
        has_resources = (self.cpu_remaining >= pod.cpu and
                         self.mem_remaining >= pod.memory)
        if not has_resources:
            return False

        # 2. Check for anti-affinity rule
        if pod.name in self.hosted_processes:
            return False

        return True

    def add_pod(self, pod: Pod):
        """Adds a pod to the node and updates resources."""
        if not self.can_fit(pod):
            raise ValueError(f"Pod {pod.name} cannot fit on Node {self.id}")

        self.pods.append(pod)
        self.cpu_remaining -= pod.cpu
        self.mem_remaining -= pod.memory
        self.hosted_processes.add(pod.name)

    def __str__(self):
        cpu_usage = (self.cpu_capacity - self.cpu_remaining) / self.cpu_capacity * 100
        mem_usage = (self.mem_capacity - self.mem_remaining) / self.mem_capacity * 100
        return (
            f"Node-{self.id}: "
            f"CPU: {self.cpu_capacity - self.cpu_remaining:.2f}/{self.cpu_capacity} ({cpu_usage:.1f}%) | "
            f"Mem: {self.mem_capacity - self.mem_remaining:.2f}/{self.mem_capacity} GiB ({mem_usage:.1f}%) | "
            f"Pods: {len(self.pods)}"
        )


def parse_workload_data(workload_csv, skip_header=True):
    """Parses CSV data into a list of Process dataclasses."""
    processes = []
    # Use StringIO to treat the string as a file
    reader = csv.reader(StringIO(workload_csv))
    if skip_header:
        _ = next(reader) # Skip header
    seen_names = defaultdict(lambda: 0)
    for row in reader:
        if not row or row[0].strip().startswith('#'): # Skip empty or commented lines
            continue
        try:
            # Create a Process object instead of a dictionary
            name = row[0].strip()
            seen_names[name] += 1
            if seen_names[name] > 1:
                # Regard same process name on multiple lines as separate processes.
                name = f"{name}[{seen_names[name]}]"
            processes.append(Process(
                name=name,
                memory=float(row[1]),
                cpu=float(row[2]),
                instances=int(row[3])
            ))
        except (ValueError, IndexError) as e:
            print(f"Warning: Skipping malformed row: {row}. Error: {e}")
    return processes


def calculate_node_requirements(processes, node_cpu, node_mem, overhead_cpu, overhead_mem):
    """
    Calculates the number of nodes required using the FFD heuristic.
    """
    allocatable_cpu = node_cpu - overhead_cpu
    allocatable_mem = node_mem - overhead_mem

    print("--- Configuration ---")
    print(f"Node Type: c5.4xlarge ({node_cpu} vCPU, {node_mem} GiB RAM)")
    print(f"System Overhead: {overhead_cpu} vCPU, {overhead_mem} GiB RAM")
    print(f"Allocatable per Node: {allocatable_cpu} vCPU, {allocatable_mem} GiB RAM\n")

    # 1. Expand processes into a flat list of individual pods
    all_pods = []
    for process in processes:
        for i in range(process.instances):
            # Add a unique identifier for clarity in logs
            pod_instance_name = f"{process.name}-{i+1}"
            all_pods.append(Pod(
                name=process.name,
                instance_name=pod_instance_name,
                memory=process.memory,
                cpu=process.cpu
            ))

    # 2. Sort pods by memory in descending order (the "Decreasing" part of FFD)
    all_pods.sort(key=lambda p: p.memory, reverse=True)

    print(f"--- Packing ---")
    print(f"Total pods to schedule: {len(all_pods)}\n")

    nodes = []
    # 3. Pack the pods
    for pod in all_pods:
        placed = False
        # Try to place in an existing node (the "First Fit" part)
        for node in nodes:
            if node.can_fit(pod):
                node.add_pod(pod)
                placed = True
                print(f"Placed {pod.instance_name} on existing Node-{node.id}")
                break

        # If it couldn't be placed, create a new node
        if not placed:
            new_node_id = len(nodes) + 1
            new_node = Node(new_node_id, allocatable_cpu, allocatable_mem)
            new_node.add_pod(pod)
            nodes.append(new_node)
            print(f"Placed {pod.instance_name} on NEW Node-{new_node.id}")

    return nodes


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"Usage: python3 {sys.argv[0]} <PROCESS_LIST_CSV>")
        sys.exit(1)

    # --- System and Node Configuration ---
    NODE_TOTAL_CPU = 16.0
    NODE_TOTAL_MEM = 32.0 # GiB

    # Define system overhead reservations
    SYSTEM_OVERHEAD_CPU = 1.0
    SYSTEM_OVERHEAD_MEM = 4.0 # GiB

    # --- Workload Definition (CSV format) ---
    # You can easily replace this with reading from a file:
    # with open('workloads.csv', 'r') as f:
    #     workload_data = f.read()
    with open(sys.argv[1]) as f:
        workload_data = f.read()

    # --- Main Execution ---
    processes = parse_workload_data(workload_data, skip_header=True)
    final_nodes = calculate_node_requirements(
        processes,
        NODE_TOTAL_CPU,
        NODE_TOTAL_MEM,
        SYSTEM_OVERHEAD_CPU,
        SYSTEM_OVERHEAD_MEM
    )

    # --- Final Report ---
    print("\n--- Final Placement Report ---")
    print(f"Total EC2 instances required: {len(final_nodes)}")
    for node in final_nodes:
        print(f"\n{node}")
        # Sort pods by name for consistent output
        sorted_pods = sorted(node.pods, key=lambda p: p.instance_name)
        for pod in sorted_pods:
            print(f"  - {pod.instance_name} (CPU: {pod.cpu}, Mem: {pod.memory})")

