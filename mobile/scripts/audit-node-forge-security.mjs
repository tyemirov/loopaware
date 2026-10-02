// @ts-check
import assert from "node:assert/strict";
import { generateKeyPairSync, verify } from "node:crypto";
import { readFileSync, realpathSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const require = createRequire(import.meta.url);
const consumers = ["@expo/cli", "@expo/code-signing-certificates"];
const nativeKeys = generateKeyPairSync("rsa", { modulusLength: 2048, publicExponent: 3 });
const pem = {
  privateKeyPEM: nativeKeys.privateKey.export({ type: "pkcs1", format: "pem" }).toString(),
  publicKeyPEM: nativeKeys.publicKey.export({ type: "pkcs1", format: "pem" }).toString(),
};
const message = "LoopAware RSA signature boundary";
const roots = new Set();

/** @typedef {{tagClass: number, type: number, constructed: boolean, value: string | Asn1[]}} Asn1 */

for (const consumer of consumers) {
  const consumerRequire = createRequire(require.resolve(`${consumer}/package.json`));
  const forge = consumerRequire("node-forge");
  const identityPath = consumerRequire.resolve("node-forge/package.json");
  const identity = JSON.parse(readFileSync(identityPath, "utf8"));
  const { asn1 } = forge;
  const keys = {
    privateKey: forge.pki.privateKeyFromPem(pem.privateKeyPEM),
    publicKey: forge.pki.publicKeyFromPem(pem.publicKeyPEM),
  };
  /** @param {Asn1[]} children @returns {Asn1} */
  const sequence = (children) => asn1.create(asn1.Class.UNIVERSAL, asn1.Type.SEQUENCE, true, children);
  /** @param {string} value @returns {Asn1} */
  const oid = (value) => asn1.create(asn1.Class.UNIVERSAL, asn1.Type.OID, false, asn1.oidToDer(value).getBytes());
  /** @param {string} value @returns {Asn1} */
  const octets = (value) => asn1.create(asn1.Class.UNIVERSAL, asn1.Type.OCTETSTRING, false, value);
  const nullParameter = (value = "") => asn1.create(asn1.Class.UNIVERSAL, asn1.Type.NULL, false, value);

  for (const algorithm of ["sha256", "md5"]) {
    const digest = forge.md[algorithm].create().update(message).digest().getBytes();
    // Sign crafted DigestInfo bytes. Verification always uses its normal public API.
    /** @param {Asn1[]} parameters @param {Asn1[]} [outerElements] */
    const signature = (parameters, outerElements = []) => keys.privateKey.sign(
      asn1.toDer(sequence([
        sequence([oid(forge.oids[algorithm]), ...parameters]), octets(digest), ...outerElements,
      ])).getBytes(), "NONE",
    );
    assert.equal(keys.publicKey.verify(digest, signature([nullParameter()])), true, `${consumer}: valid ${algorithm}`);
    if (algorithm === "sha256") {
      assert.equal(keys.publicKey.verify(digest, signature([])), true, `${consumer}: SHA-256 without NULL`);
    } else {
      assert.throws(() => keys.publicKey.verify(digest, signature([])), /Missing algorithm identifier NULL parameters/);
    }
    /** @type {[string, Asn1[], Asn1[]][]} */
    const malformed = [
      ["extra nested element after NULL", [nullParameter(), octets("garbage")], []],
      ["extra nested element without NULL", [octets("garbage")], []],
      ["nonempty NULL", [nullParameter("garbage")], []],
      ["extra outer element", [nullParameter()], [octets("garbage")]],
    ];
    for (const [name, parameters, outerElements] of malformed) {
      assert.throws(
        () => keys.publicKey.verify(digest, signature(parameters, outerElements)),
        /ASN\.1 object does not contain a valid RSASSA-PKCS1-v1_5 DigestInfo value/,
        `${consumer}: ${algorithm} accepted ${name}`,
      );
    }
    const wrongDigest = forge.md[algorithm].create().update(`${message} changed`).digest().getBytes();
    assert.equal(keys.publicKey.verify(wrongDigest, signature([nullParameter()])), false);
  }

  assert.equal(identity.name, "@loopaware/node-forge", `${consumer}: uncorrected package identity`);
  assert.equal(identity.version, "1.4.0-loopaware.1");
  roots.add(realpathSync(path.dirname(identityPath)));
}
assert.equal(roots.size, 1, "Expo consumers must use one corrected Forge implementation");

const certificates = require("@expo/code-signing-certificates");
const keys = certificates.convertKeyPairPEMToKeyPair(pem);
const now = Date.now();
const certificate = certificates.generateSelfSignedCodeSigningCertificate({
  keyPair: keys,
  validityNotBefore: new Date(now - 60_000),
  validityNotAfter: new Date(now + 3_600_000),
  commonName: "LoopAware security test",
});
const parsed = certificates.convertCertificatePEMToCertificate(certificates.convertCertificateToCertificatePEM(certificate));
certificates.validateSelfSignedCertificate(parsed, keys);
const signed = certificates.signBufferRSASHA256AndVerify(keys.privateKey, parsed, Buffer.from(message));
assert.equal(verify("sha256", Buffer.from(message), nativeKeys.publicKey, Buffer.from(signed, "base64")), true);
assert.equal(certificates.convertCSRPEMToCSR(certificates.convertCSRToCSRPEM(certificates.generateCSR(keys, "LoopAware test"))).verify(), true);
console.log("mobile_node_forge_security.ok: both Expo consumers reject malformed signatures and preserve certificate operations");
