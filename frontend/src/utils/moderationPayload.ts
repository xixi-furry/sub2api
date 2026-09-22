// Optional request-format example. All audit instructions belong in the system prompt.
export const moderationPayloadExample = `const wrappedUserContent =
  "<user_input>\\n" +
  text.replace(/</g, "&lt;").replace(/>/g, "&gt;") +
  "\\n</user_input>";

const userContent = Array.isArray(input)
  ? [{ type: "text", text: wrappedUserContent },
     ...input.filter(part => part.type === "image_url")]
  : wrappedUserContent;

const requestBody = isModerationEndpoint
  ? JSON.stringify({ model: config.model, input })
  : JSON.stringify({
      model: config.model,
      messages: [
        { role: "system", content: config.auditPrompt },
        { role: "user", content: userContent },
      ],
      temperature: 0,
      stream: false,
    });`
