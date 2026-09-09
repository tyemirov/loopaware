// @ts-check
const { withAppBuildGradle, withMainApplication, withXcodeProject } = require('expo/config-plugins');

/** Generate store projects with embedded JavaScript bundles.
 * @param {import("expo/config").ExpoConfig} config
 */
module.exports = function withStoreBuild(config) {
    config = withAppBuildGradle(config, (project) => {
        const source = project.modResults.contents;
        const expoCli = /cliFile = new File\([^\n]+\n\s*bundleCommand = "export:embed"/;
        if (!expoCli.test(source)) throw new Error('Android native bundle configuration changed.');
        project.modResults.contents = source.replace(expoCli,
            'cliFile = new File(["node", "--print", "require(\'@react-native-community/cli\').bin"].execute(null, rootDir).text.trim())\n    bundleCommand = "bundle"\n    debuggableVariants = []');
        const releaseOffset = project.modResults.contents.indexOf('        release {', project.modResults.contents.indexOf('buildTypes {'));
        if (releaseOffset < 0) throw new Error('Android release configuration is missing.');
        project.modResults.contents = project.modResults.contents.slice(0, releaseOffset) + project.modResults.contents.slice(releaseOffset).replace('signingConfig signingConfigs.debug', 'signingConfig signingConfigs.release');
        project.modResults.contents = project.modResults.contents.replace('signingConfigs {', `signingConfigs {
        release {
            storeFile file(System.getenv("LOOPAWARE_ANDROID_KEYSTORE") ?: "missing-release-keystore")
            storePassword System.getenv("LOOPAWARE_ANDROID_STORE_PASSWORD") ?: ""
            keyAlias System.getenv("LOOPAWARE_ANDROID_KEY_ALIAS") ?: ""
            keyPassword System.getenv("LOOPAWARE_ANDROID_KEY_PASSWORD") ?: ""
        }`);
        project.modResults.contents = project.modResults.contents.replace(/versionCode \d+/, 'versionCode (System.getenv("MPRLAB_MOBILE_VERSION_CODE") ?: "1").toInteger()').replace(/versionName "[^"]+"/, 'versionName System.getenv("MPRLAB_MOBILE_VERSION_NAME") ?: "2026.6.19"');
        project.modResults.contents = project.modResults.contents.replace(/def enableMinifyInReleaseBuilds = [^\n]+/, 'def enableMinifyInReleaseBuilds = true');
        return project;
    });
    config = withMainApplication(config, (project) => {
        const contextArgument = 'context = applicationContext,';
        if (!project.modResults.contents.includes(contextArgument)) throw new Error('Android developer support configuration changed.');
        project.modResults.contents = project.modResults.contents.replace(contextArgument, `${contextArgument}\n      useDevSupport = false,`);
        return project;
    });
    return withXcodeProject(config, (project) => {
        const configurations = Object.entries(project.modResults.pbxXCBuildConfigurationSection())
            .filter(([id, value]) => !id.endsWith('_comment') && String(value.buildSettings?.PRODUCT_BUNDLE_IDENTIFIER).replace(/^"|"$/g, '') === config.ios?.bundleIdentifier);
        if (configurations.length === 0) throw new Error('The Apple application build configurations are missing.');
        for (const [, value] of configurations) value.buildSettings.CODE_SIGN_STYLE = 'Automatic';
        const phases = project.modResults.hash.project.objects.PBXShellScriptBuildPhase;
        let updatedPhase = false;
        for (const phase of Object.values(phases)) {
            if (typeof phase !== 'object' || !phase.shellScript?.includes('react-native-xcode.sh')) continue;
            // Use the native React Native bundler for artifacts; Expo remains the source-config generator.
            const script = 'set -e\nPROJECT_ROOT="$(cd "$PROJECT_DIR/.." && pwd -P)"\nexport PROJECT_ROOT\nexport ENTRY_FILE="$PROJECT_ROOT/index.ts"\nexport CLI_PATH="$PROJECT_ROOT/node_modules/react-native/scripts/bundle.js"\nexport BUNDLE_COMMAND=bundle\nexport FORCE_BUNDLING=1\nexport NODE_BINARY="$(command -v node)"\nif [ -f "$PROJECT_DIR/.xcode.env.local" ]; then\n  . "$PROJECT_DIR/.xcode.env.local"\nfi\n/bin/sh "$PROJECT_DIR/../node_modules/react-native/scripts/react-native-xcode.sh"\n';
            phase.shellScript = JSON.stringify(script);
            updatedPhase = true;
        }
        if (!updatedPhase) throw new Error('iOS native bundle phase is missing.');
        return project;
    });
};
