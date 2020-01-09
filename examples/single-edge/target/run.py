"""
Applies a user-provided function to the files in the input directory one by one, aggregates the
results in memory (in a streaming fashion), and writes the output to the result file in its output
directory
"""

def reducer(accumulator, value):
    """
    Reducer function to apply to the files in the input directory

    Args:
    accumulator
        Dictionary with two keys - 'sum', 'count'. 'sum' maintains a running sum of the values and
        'count' maintains a counter of the number of values seen. These can be used to, for example,
        calculate effectively the streaming mean of the input files
    value
        Value in current input file
    """
    try:
        value = float(value)
    except:
        accumulator['errors'] = accumulator.get('errors', 0) + 1
        return accumulator
    return {
        'sum': accumulator.get('sum', 0.0) + float(value),
        'count': accumulator.get('count', 0) + 1,
        'errors': accumulator.get('errors', 0),
    }

if __name__ == '__main__':
    import json
    import os

    INPUT_DIR = os.environ.get('INPUT_DIR', '/simplex/inputs')
    OUTPUT_DIR = os.environ.get('OUTPUT_DIR', '/simplex/outputs')

    accumulator = {}

    for filename in os.listdir(INPUT_DIR):
        filepath = os.path.join(INPUT_DIR, filename)
        if os.path.isfile(filepath):
            with open(filepath, 'r') as ifp:
                contents = ifp.read()
            accumulator = reducer(accumulator, contents)

    with open(os.path.join(OUTPUT_DIR, 'result'), 'w') as ofp:
        json.dump(accumulator, ofp)
