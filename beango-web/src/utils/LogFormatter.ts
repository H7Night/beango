export const formatLogOutput = (data: any): string => {
  return JSON.stringify(data, null, 2);
};

export const highlightJSON = (jsonString: string): string => {
  if (!jsonString) return "";
  let json = jsonString
    .replace(/(&)/g, "&amp;")
    .replace(/(>)/g, "&gt;")
    .replace(/(<)/g, "&lt;")
    .replace(
      /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
      (match) => {
        let cls = "number";
        if (/^"/.test(match)) {
          if (/:$/.test(match)) {
            cls = "key";
          } else {
            cls = "string";
          }
        } else if (/true|false/.test(match)) {
          cls = "boolean";
        } else if (/null/.test(match)) {
          cls = "null";
        }
        return `<span class="${cls}">${match}</span>`;
      }
    );
  return json;
};