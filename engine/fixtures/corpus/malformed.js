// Malformed JS syntax to test error recovery
function incomplete( {
  const x = ;
  const unclosed = "unterminated string;
  if (
    return ;
  }
}
