import * as path from "path";
import Mocha = require("mocha");
import * as fs from "fs";

export function run(): Promise<void> {
  // Create the mocha test
  const mocha = new Mocha({
    ui: "bdd",
    color: true,
    timeout: 10000,
  });

  const testsRoot = path.resolve(__dirname, ".");

  return new Promise((resolve, reject) => {
    try {
      // Read test files directly
      const files = fs
        .readdirSync(testsRoot)
        .filter((file: string) => file.endsWith(".test.js"))
        .sort();

      // Add files to the test suite
      files.forEach((file: string) => {
        mocha.addFile(path.resolve(testsRoot, file));
      });

      // Run the mocha test
      mocha.run((failures: number) => {
        if (failures > 0) {
          reject(new Error(`${failures} tests failed.`));
        } else {
          resolve();
        }
      });
    } catch (err) {
      console.error(err);
      reject(err);
    }
  });
}
