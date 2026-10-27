// @ts-check
import { readFile, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const mobile = resolve(import.meta.dirname, '..');
const wrapperPath = resolve(mobile, 'android/gradlew');
const propertiesPath = resolve(mobile, 'android/gradle.properties');
const wrapperMarker = '# Collect all arguments for the java command:\n';
const compilerPolicy = 'set -- "-Pkotlin.compiler.execution.strategy=in-process" "$@"\n\n';

const wrapper = await readFile(wrapperPath, 'utf8');
const occurrences = wrapper.split(compilerPolicy).length - 1;
if (occurrences > 1 || !wrapper.includes(wrapperMarker)) throw new Error('Android Gradle wrapper policy boundary changed.');
const preparedWrapper = occurrences === 1 ? wrapper : wrapper.replace(wrapperMarker, compilerPolicy + wrapperMarker);

const properties = await readFile(propertiesPath, 'utf8');
const jvmArguments = [...properties.matchAll(/^org\.gradle\.jvmargs=(.*)$/gm)];
if (jvmArguments.length !== 1 || (jvmArguments[0][1].match(/-XX:MaxMetaspaceSize=\S+/g) ?? []).length !== 1) {
    throw new Error('Android Gradle JVM arguments are missing or ambiguous.');
}
const selectedArguments = jvmArguments[0][1].replace(/-XX:MaxMetaspaceSize=\S+/, '-XX:MaxMetaspaceSize=1g');
const preparedProperties = properties.replace(/^org\.gradle\.jvmargs=.*$/m, 'org.gradle.jvmargs=' + selectedArguments);
if (preparedWrapper !== wrapper) await writeFile(wrapperPath, preparedWrapper);
if (preparedProperties !== properties) await writeFile(propertiesPath, preparedProperties);
console.info('android_build_policy.prepared');
