// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export default class Validator {
    constructor() {
        // Our list of components we have to validate before allowing a submit action.
        this.components = new Map();
    }

    addComponent = (key, ref) => {
        this.components.set(key, ref);
    };

    removeComponent = (key) => {
        this.components.delete(key);
    };

    validate = () => {
        const validator = (accum, ref) => {
            let currentRef = ref.current;

            // If the ref was wrapped by react-redux connect, unwrap it
            if (typeof currentRef.getWrappedInstance === 'function') {
                currentRef = currentRef.getWrappedInstance();
            }

            // Check every field, but only return true if every field is valid.
            return currentRef.isValid() && accum;
        };
        return Array.from(this.components.values()).reduce(validator, true);
    };
}
