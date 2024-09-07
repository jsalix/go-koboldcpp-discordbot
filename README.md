## go-koboldcpp-discordbot

Super basic, proof-of-concept Go script to make a locally hosted LLM (in this case using https://github.com/LostRuins/koboldcpp) available as a convenient Discord bot (the prompting/context system is very "simple" and relies heavily on the LLM)

required `DISCORD_TOKEN`, `API_URL`, `PERSONA` go in a new `.env` file:

```.env
DISCORD_TOKEN=
API_URL=http://localhost:5001/api
PERSONA=Kobold
```

```shell
go run .
```

**experimental, more like an example, made for personal use**