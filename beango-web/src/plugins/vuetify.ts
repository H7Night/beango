import {createVuetify} from "vuetify/framework";
import 'vuetify/_styles.scss'
import {components, directives} from "vuetify/lib/entry-bundler";

export default createVuetify({
    components,
    directives,
    icons:{
        defaultSet:'mdi',
    },
})