async function o(){const t=await fetch("/api/automation");if(!t.ok)throw new Error(`Failed to fetch automations: ${t.status}`);return(await t.json()).items??[]}export{o as g};
