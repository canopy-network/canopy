import { objEmpty } from "@/components/util";
import { useRef, useState } from "react";
import { CopyIcon } from "@/components/svg_icons";

// A light neat stylable JSON viewer
function CanaJSON({ state, setState, JsonViewVariant }) {
  const isEmptyPK = objEmpty(state.pk);
  const isEmptyTxRes = objEmpty(state.txResult);
  const needsWrapper = !isEmptyTxRes && Object.keys(state.txResult).length === 0;
  const [copySuccess, setCopySuccess] = useState(false);
  const jsonRef = useRef(null);
  if (isEmptyPK && isEmptyTxRes) return <></>;

  let formattedJson = isEmptyPK ? state.txResult : state.pk;
  // if (needsWrapper) {
  //   formattedJson = { result: formattedJson}
  // }

  const handleCopyClick = (event) => {
    event.preventDefault();
    if (jsonRef.current) {
      const jsonString = jsonRef.current.innerText;
      navigator.clipboard
        .writeText(jsonString)
        .then(() => {
          setCopySuccess(true);
          setState({ ...state, toast: "JSON Copied to Clipboard!" });
          setTimeout(() => setCopySuccess(false), 2000);
        })
        .catch((err) => {
          console.error("Failed to copy: ", err);
          setState({ ...state, toast: "Failed to Copy JSON." });
        });
    }
  };

  const renderJson = (json, level = 0) => {
    if (typeof json === "object" && json !== null) {
      const keys = Object.keys(json);
      return (
        <>
          {level > 0 && "{"}
          {keys.map((key, index) => (
            <div key={key} className="json-entry">
              <span className="json-key">"{key}": </span>
              {renderJson(json[key], level + 1)}
              {index < keys.length - 1 && ","}
            </div>
          ))}
          {level > 0 && "}"}
        </>
      );
    } else if (typeof json === "number" || typeof json === "boolean") {
      return <span className={`json-value ${typeof json}`}>{json}</span>;
    } else {
      return <span className="json-value string">"{json}"</span>;
    }
  };

  return (
    <div className={`json-viewer ${JsonViewVariant || ""}`}>
      <pre ref={jsonRef}>
        <div className="json-entry">{renderJson(formattedJson, 1)}</div>
      </pre>
      <a href="#" onClick={handleCopyClick} className="copy-link">
        {copySuccess ? <CopyIcon /> : <CopyIcon />}
      </a>
    </div>
  );
}

export default CanaJSON;
