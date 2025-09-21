import yaml
import json
import sys


def main():
    filename = "playground.yml"
    
    try:
        with open(filename, 'r') as file:
            # safe_load_all returns a generator. We convert it to a list
            # to capture all documents from the file.
            for i, doc in enumerate(yaml.safe_load_all(file)):
                print(f"--- Document #{i+1} ---")
                json_output = json.dumps(doc, indent=2)
                print(json_output)

    except FileNotFoundError:
        print(f"Error: File '{filename}' not found.", file=sys.stderr)
    except yaml.YAMLError as e:
        print(f"Error parsing YAML file: {e}", file=sys.stderr)


if __name__ == "__main__":
    main()

