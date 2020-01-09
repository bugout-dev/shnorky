"""
A simple script that materializes files to be processed in its output directory.

Although this script creates the files from thin air, it is meant to simulate scripts that produce
the files from object storage or as a result of running database queries.
"""

import os
import random
import uuid

OUTPUT_DIR = os.environ.get('OUTPUT_DIR', '/simplex/outputs')
NUM_SAMPLES= int(os.environ.get('NUM_SAMPLES', '100'))

for _ in range(NUM_SAMPLES):
    filename = str(uuid.uuid4())
    output_path = os.path.join(OUTPUT_DIR, filename)
    contents = random.random()
    with open(output_path, 'w') as ofp:
        ofp.write(str(contents))
