import React, {PureComponent} from 'react';
import {FixedSizeList} from 'react-window';
import PropTypes from 'prop-types';

// height of a react select option
const optionHeight = 35;

class VirtualizedList extends PureComponent {
    static propTypes = {
        options: PropTypes.array.isRequired,
        children: PropTypes.array.isRequired,
        maxHeight: PropTypes.number.isRequired,
        getValue: PropTypes.func.isRequired,
    }

    render() {
        const {options, children, maxHeight, getValue} = this.props;
        const value = getValue()[0];
        const initialOffset = options.indexOf(value) * optionHeight;

        return (
            <FixedSizeList
                height={maxHeight}
                itemCount={children.length}
                itemSize={optionHeight}
                initialScrollOffset={initialOffset}
            >
                {(data) => (
                    <div style={data.style}>{children[data.index]}</div>
                )}
            </FixedSizeList>
        );
    }
}

export default VirtualizedList;
