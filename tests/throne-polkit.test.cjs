const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

let rule;
vm.runInNewContext(
  fs.readFileSync(path.join(__dirname, "../nix/throne-tun.rules"), "utf8"),
  {
    polkit: {
      addRule: (callback) => { rule = callback; },
      Result: { YES: "yes" },
    },
  },
);

for (const action of ["revert", "set-domains", "set-default-route", "set-dns-servers"]) {
  for (const member of [true, false]) {
    test(`${action}, group member=${member}`, () => {
      const result = rule(
        { id: `org.freedesktop.resolve1.${action}` },
        { isInGroup: (group) => member && group === "throne-tun" },
      );
      assert.equal(result, member ? "yes" : undefined);
    });
  }
}

test("does not authorize unrelated privileged actions", () => {
  assert.equal(
    rule({ id: "org.freedesktop.systemd1.manage-units" }, { isInGroup: () => true }),
    undefined,
  );
});
