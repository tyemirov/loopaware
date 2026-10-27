// @ts-check
import { resolve } from 'node:path';
import { verifyPreparation } from '../../mobile/scripts/record-store-preparation.mjs';
verifyPreparation(resolve('mobile/prepared'));
console.info('Selected public native inputs and semantic preparation facts are available for the release.');
