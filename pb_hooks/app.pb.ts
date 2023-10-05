
// Extending PocketBase with JS - @see https://pocketbase.io/docs/js-overview/

/// <reference path="../pb_data/types.d.ts" />

routerAdd("GET", "/app/:name", (c) => {
    let name = c.pathParam("name")

    return c.json(200, { "message": "Hello " + name })
})
//
onModelAfterUpdate((e) => {
    console.log("user updated...", e.model.get("title"))
}, "website")