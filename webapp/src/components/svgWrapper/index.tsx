import React from 'react';

type SVGWrapperProps = {
    viewBox?: string;
    height?: number;
    width?: number;
    fill?: string;
    onHoverFill?: string;
    children: React.ReactNode;
    className?: string;
}

const SVGWrapper = ({
    children,
    viewBox = '0 0 36 36',
    height = 36,
    width = 36,
    fill = 'none',
    onHoverFill,
    className = '',
}: SVGWrapperProps): JSX.Element => {
    return (
        <svg
            width={width}
            height={height}
            viewBox={viewBox}
            fill={onHoverFill || fill}
            xmlns='http://www.w3.org/2000/svg'
            className={className}
        >
            {children}
        </svg>
    );
};

export default SVGWrapper;
