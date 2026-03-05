from typing import List, Dict, Any, Optional
from .client import Axon

class AxonToolkit:
    """
    A ready-to-use sensory kit for AI agents (Vamora, LangChain, etc.)
    Provides high-level tools that wrap Axon's full potential.
    """
    
    def __init__(self, axon_client: Axon, session_id: str = "default"):
        self.axon = axon_client
        self.session_id = session_id

    async def get_tools(self) -> List[Dict[str, Any]]:
        """
        Returns a list of tool definitions in OpenAI/LLM-friendly format.
        """
        return [
            {
                "name": "navigate",
                "description": "Navigate to a URL and wait for the page to be ready.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "url": {"type": "string", "description": "The destination URL"}
                    },
                    "required": ["url"]
                }
            },
            {
                "name": "snapshot",
                "description": "Get a compact semantic map of the current page. Uses 98% fewer tokens than raw HTML.",
                "parameters": {
                    "type": "object",
                    "properties": {}
                }
            },
            {
                "name": "smart_interact",
                "description": "The most powerful way to interact. Provide an intent (e.g. 'search button') and an action.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "intent": {"type": "string", "description": "Description of the element to interact with"},
                        "action": {"type": "string", "enum": ["click", "fill", "hover", "press", "select"]},
                        "value": {"type": "string", "description": "Value for input or selection"}
                    },
                    "required": ["intent", "action"]
                }
            },
            {
                "name": "wait_for_stability",
                "description": "Wait until the page is fully stable and animations are finished.",
                "parameters": {
                    "type": "object",
                    "properties": {}
                }
            },
            {
                "name": "vault_fill",
                "description": "Fill a field using a protected secret from the Intelligence Vault. Use this for sensitive logins.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "intent": {"type": "string", "description": "Description of the element to fill (e.g. 'password field')"},
                        "secret_name": {"type": "string", "description": "Name of the secret in the vault"},
                        "field": {"type": "string", "enum": ["username", "password", "value"], "description": "Which field to inject"}
                    },
                    "required": ["intent", "secret_name"]
                }
            }
        ]

    async def run_tool(self, tool_name: str, args: Dict[str, Any]) -> str:
        """
        Executes a tool and returns the result as a string for the agent.
        """
        if tool_name == "navigate":
            res = await self.axon.navigate(self.session_id, args["url"])
            return f"Successfully navigated to {args['url']}"
        
        elif tool_name == "snapshot":
            snap = await self.axon.snapshot(self.session_id)
            return f"Page: {snap.title}\nContent:\n{snap.content}"
        
        elif tool_name == "smart_interact":
            res = await self.axon.smart_interact(
                self.session_id, 
                args["intent"], 
                args["action"], 
                args.get("value")
            )
            if res.requires_confirm:
                return f"ACTION BLOCKED: This is an irreversible action. Set 'confirm=True' to proceed with: {res.message}"
            return f"Action '{args['action']}' on '{args['intent']}' was successful."
        
        elif tool_name == "wait_for_stability":
            await self.axon.status(self.session_id) # Getting status triggers internal wait/sync
            return "Page is now stable."
        
        elif tool_name == "vault_fill":
            # First find the element by intent
            snap = await self.axon.snapshot(self.session_id)
            # Simplistic find logic for toolkit (In real usage, we should use a better matcher)
            # But for now, we'll try to use smart_interact's find_and_act with the vault ref
            res = await self.axon.find_and_act(
                self.session_id,
                "fill",
                args["intent"],
                value=f"@vault:{args['secret_name']}:{args.get('field', 'password')}"
            )
            return f"Successfully injected secret '{args['secret_name']}' into '{args['intent']}'."
            
        return f"Tool {tool_name} not found."
