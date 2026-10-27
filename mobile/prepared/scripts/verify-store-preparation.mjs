// @ts-check
import { verifyPreparation } from './record-store-preparation.mjs';
verifyPreparation(process.cwd());
console.info('Prepared native application configuration and dependency versions match the current semantic record.');
