import { WASI } from "@runno/wasi";

export const runModule = async (
  moduleURL: string,
  stdin: string
): Promise<string> => {
  let resp = "";

  let done = false;
  const result = WASI.start(fetch(moduleURL), {
    args: ["main.wasm"],
    stdout: (out: string) => resp += out,
    stderr: (err: string) => console.error("stderr", err),
    stdin: () => {
      if (!done) {
        done = true
        return stdin
      } else {
        return null
      }
    },
   
  });

  await result;

  return resp;
};

const btn = document.querySelector("#run") as HTMLButtonElement
const codeArea = document.querySelector("#code") as HTMLTextAreaElement
const outputArea = document.querySelector("#output") as HTMLDivElement

btn.onclick = async () => {
  const code: string = codeArea.value

  console.log("running")
  const bc = await runModule("compiler.wasm", code)

  console.log({bc})

  const out = await runModule("interpreter.wasm", bc)
  console.log(out)

  outputArea.innerText = out
}
  
